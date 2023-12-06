package appgateserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/client"
	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// SigningInformation is a struct that holds information related to the signing
// of relay requests, used by the appGateServer to determine how they will sign
// relay requests (with either their own ring or the rign of the application).
type SigningInformation struct {
	// SelfSigning indicates whether the server is running in self-signing mode
	SelfSigning bool

	// SigningKeyName is the name of the key in the keyring that corresponds to the
	// private key used to sign relay requests.
	SigningKeyName string

	// SigningKey is the scalar point on the appropriate curve corresponding to the
	// signer's private key, and is used to sign relay requests via a ring signature
	SigningKey ringtypes.Scalar

	// AppAddress is the address of the application that the server is serving if
	// If it is nil, then the application address must be included in each request via a query parameter.
	AppAddress string
}

// appGateServer is the server that listens for application requests and relays them to the supplier.
// It is responsible for maintaining the current session for the application, signing the requests,
// and verifying the response signatures.
// The appGateServer is the basis for both applications and gateways, depending on whether the application
// is running their own instance of the appGateServer or they are sending requests to a gateway running an
// instance of the appGateServer, they will need to either include the application address in the request or not.
type appGateServer struct {
	logger polylog.Logger

	// signing information holds the signing key and application address for the server
	signingInformation *SigningInformation

	// ringCache is used to obtain and store the ring for the application.
	ringCache crypto.RingCache

	// clientCtx is the client context for the application.
	// It is used to query for the application's account to unmarshal the supplier's account
	// and get the public key to verify the relay response signature.
	clientCtx querytypes.Context

	// sessionQuerier is the querier for the session module.
	// It used to get the current session for the application given a requested service.
	sessionQuerier sessiontypes.QueryClient

	// sessionMu is a mutex to protect currentSession map reads and and updates.
	sessionMu sync.RWMutex

	// currentSessions is the current session for the application given a block height.
	// It is updated by the goListenForNewSessions goroutine.
	currentSessions map[string]*sessiontypes.Session

	// accountQuerier is the querier for the account module.
	// It is used to get the the supplier's public key to verify the relay response signature.
	accountQuerier client.AccountQueryClient

	// blockClient is the client for the block module.
	// It is used to get the current block height to query for the current session.
	blockClient client.BlockClient

	// listeningEndpoint is the endpoint that the appGateServer will listen on.
	listeningEndpoint *url.URL

	// server is the HTTP server that will be used capture application requests
	// so that they can be signed and relayed to the supplier.
	server *http.Server

	// accountCache is a cache of the supplier accounts that has been queried
	// TODO_TECHDEBT: Add a size limit to the cache.
	supplierAccountCache map[string]cryptotypes.PubKey
}

// NewAppGateServer creates a new appGateServer instance with the given dependencies.
//
// Required dependencies:
// - polylog.Logger
// - sdkclient.Context
// - client.BlockClient
// - client.AccountQueryClient
// - crypto.RingCache
func NewAppGateServer(
	deps depinject.Config,
	opts ...appGateServerOption,
) (*appGateServer, error) {
	app := &appGateServer{
		currentSessions:      make(map[string]*sessiontypes.Session),
		supplierAccountCache: make(map[string]cryptotypes.PubKey),
	}

	if err := depinject.Inject(
		deps,
		&app.logger,
		&app.clientCtx,
		&app.blockClient,
		&app.accountQuerier,
		&app.ringCache,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(app)
	}

	if err := app.validateConfig(); err != nil {
		return nil, err
	}

	keyRecord, err := app.clientCtx.Keyring.Key(app.signingInformation.SigningKeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key from keyring: %w", err)
	}

	appAddress, err := keyRecord.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address from key: %w", err)
	}
	if app.signingInformation.SelfSigning {
		app.signingInformation.AppAddress = appAddress.String()
	}

	// Convert the key record to a private key and return the scalar
	// point on the secp256k1 curve that it corresponds to.
	// If the key is not a secp256k1 key, this will return an error.
	signingKey, err := recordLocalToScalar(keyRecord.GetLocal())
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key to scalar: %w", err)
	}
	app.signingInformation.SigningKey = signingKey

	clientCtx := cosmosclient.Context(app.clientCtx)

	app.sessionQuerier = sessiontypes.NewQueryClient(clientCtx)
	app.server = &http.Server{Addr: app.listeningEndpoint.Host}

	return app, nil
}

// Start starts the appgate server and blocks until the context is done
// or the server returns an error.
func (app *appGateServer) Start(ctx context.Context) error {
	// Shutdown the HTTP server when the context is done.
	go func() {
		<-ctx.Done()
		app.server.Shutdown(ctx)
	}()

	// Set the HTTP handler.
	app.server.Handler = app

	// Start the HTTP server.
	return app.server.ListenAndServe()
}

// Stop stops the appgate server and returns any error that occurred.
func (app *appGateServer) Stop(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

// ServeHTTP is the HTTP handler for the appgate server.
// It captures the application request, signs it, and sends it to the supplier.
// After receiving the response from the supplier, it verifies the response signature
// before returning the response to the application.
// The serviceId is extracted from the request path.
// The request's path should be of the form:
//
//	"<protocol>://host:port/serviceId[/other/path/segments]?senderAddr=<senderAddr>"
//
// where the serviceId is the id of the service that the application is requesting
// and the other (possible) path segments are the JSON RPC request path.
// TODO_TECHDEBT: Revisit the requestPath above based on the SDK that'll be exposed in the future.
func (app *appGateServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := app.logger.WithContext(request.Context())

	// Extract the serviceId from the request path.
	path := request.URL.Path
	serviceId := strings.Split(path, "/")[1]

	// Read the request body bytes.
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		app.replyWithError(
			ctx,
			payloadBz,
			writer,
			ErrAppGateHandleRelay.Wrapf("reading relay request body: %s", err),
		)
		// TODO_TECHDEBT: log additional info?
		app.logger.Error().Err(err).Msg("failed reading relay request body")
		return
	}
	app.logger.Debug().
		Str("service_id", serviceId).
		Str("payload", string(payloadBz)).
		Msg("handling relay")

	// Determine the application address.
	appAddress := app.signingInformation.AppAddress
	if appAddress == "" {
		appAddress = request.URL.Query().Get("senderAddr")
	}
	if appAddress == "" {
		app.replyWithError(ctx, payloadBz, writer, ErrAppGateMissingAppAddress)
		// TODO_TECHDEBT: log additional info?
		app.logger.Error().Msg("no application address provided")
	}

	// TODO(@h5law, @red0ne): Add support for asynchronous relays, and switch on
	// the request type here.
	// TODO_RESEARCH: Should this be started in a goroutine, to allow for
	// concurrent requests from numerous applications?
	if err := app.handleSynchronousRelay(ctx, appAddress, serviceId, payloadBz, request, writer); err != nil {
		// Reply with an error response if there was an error handling the relay.
		app.replyWithError(ctx, payloadBz, writer, err)
		// TODO_TECHDEBT: log additional info?
		app.logger.Error().Err(err).Msg("failed handling relay")
		return
	}

	// TODO_TECHDEBT: log additional info?
	app.logger.Info().Msg("request serviced successfully")
}

// validateConfig validates the appGateServer configuration.
func (app *appGateServer) validateConfig() error {
	if app.signingInformation == nil {
		return ErrAppGateMissingSigningInformation
	}
	if app.listeningEndpoint == nil {
		return ErrAppGateMissingListeningEndpoint
	}
	return nil
}

// recordLocalToScalar converts the private key obtained from a
// key record to a scalar point on the secp256k1 curve
func recordLocalToScalar(local *keyring.Record_Local) (ringtypes.Scalar, error) {
	if local == nil {
		return nil, fmt.Errorf("cannot extract private key from key record: nil")
	}
	priv, ok := local.PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	if !ok {
		return nil, fmt.Errorf("cannot extract private key from key record: %T", local.PrivKey.GetCachedValue())
	}
	if _, ok := priv.(*secp256k1.PrivKey); !ok {
		return nil, fmt.Errorf("unexpected private key type: %T, want %T", priv, &secp256k1.PrivKey{})
	}
	crv := ring_secp256k1.NewCurve()
	privKey, err := crv.DecodeToScalar(priv.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}
	return privKey, nil
}

type appGateServerOption func(*appGateServer)
