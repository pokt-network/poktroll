package appgateserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	blocktypes "github.com/pokt-network/poktroll/pkg/client"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// appGateServer is the server that listens for application requests and relays them to the supplier.
// it is responsible for maintaining the current session for the application, signing the requests,
// and verifying the response signatures.
// The appGateServer is the basis for both applications and gateways, depending on whether the application
// is running their own instance of the appGateServer or they are sending requests to a gateway running an
// instance of the appGateServer, they will need to either include the application address in the request or not.
type appGateServer struct {
	// signingKeyName is the name of the key that will be used to sign relays
	signingKeyName string

	// signingKey is the scalar point on the appropriate curve corresponding to the
	// signer's private key, and is used to sign relay requests via a ring signature
	signingKey ringtypes.Scalar

	// ringCache is a cache of the public keys used to create the ring for a given application
	// they are stored in a map of application address to a slice of points on the secp256k1 curve
	// TODO(@h5law): subscribe to on-chain events to update this cache as the ring changes over time
	ringCache      map[string][]ringtypes.Point
	ringCacheMutex *sync.RWMutex

	// appAddress is the address of the application that the server is serving if
	// it is nil then the application address must be included in each request
	appAddress string

	// clientCtx is the client context for the application.
	// It is used to query for the application's account to unmarshal the supplier's account
	// and get the public key to verify the relay response signature.
	clientCtx sdkclient.Context

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
	accountQuerier accounttypes.QueryClient

	// applicationQuerier is the querier for the application module.
	// It is used to get the ring for a given application address.
	applicationQuerier apptypes.QueryClient

	// blockClient is the client for the block module.
	// It is used to get the current block height to query for the current session.
	blockClient blocktypes.BlockClient

	// listeningEndpoint is the endpoint that the appGateServer will listen on.
	listeningEndpoint *url.URL

	// server is the HTTP server that will be used capture application requests
	// so that they can be signed and relayed to the supplier.
	server *http.Server

	// accountCache is a cache of the supplier accounts that has been queried
	// TODO_TECHDEBT: Add a size limit to the cache.
	supplierAccountCache map[string]cryptotypes.PubKey
}

func NewAppGateServer(
	deps depinject.Config,
	opts ...appGateServerOption,
) (*appGateServer, error) {
	app := &appGateServer{
		ringCacheMutex:       &sync.RWMutex{},
		ringCache:            make(map[string][]ringtypes.Point),
		currentSessions:      make(map[string]*sessiontypes.Session),
		supplierAccountCache: make(map[string]cryptotypes.PubKey),
	}

	if err := depinject.Inject(
		deps,
		&app.clientCtx,
		&app.blockClient,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(app)
	}

	if err := app.validateConfig(); err != nil {
		return nil, err
	}

	key, err := app.clientCtx.Keyring.Key(app.signingKeyName)
	if err != nil {
		return nil, err
	}

	appAddress, err := key.GetAddress()
	if err != nil {
		return nil, err
	}
	app.appAddress = appAddress.String()

	signingKey, err := recordLocalToScalar(key.GetLocal())
	if err != nil {
		return nil, err
	}
	app.signingKey = signingKey

	app.sessionQuerier = sessiontypes.NewQueryClient(app.clientCtx)
	app.accountQuerier = accounttypes.NewQueryClient(app.clientCtx)
	app.applicationQuerier = apptypes.NewQueryClient(app.clientCtx)
	app.server = &http.Server{Addr: app.listeningEndpoint.Host}

	return app, nil
}

// Start starts the application server and blocks until the context is done
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

// Stop stops the application server and returns any error that occurred.
func (app *appGateServer) Stop(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

// ServeHTTP is the HTTP handler for the application server.
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
func (app *appGateServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	// Extract the serviceId from the request path.
	path := request.URL.Path
	serviceId := strings.Split(path, "/")[1]
	var appAddress string

	if app.appAddress != "" {
		appAddress = request.URL.Query().Get("senderAddr")
	} else {
		appAddress = app.appAddress
	}

	if appAddress == "" {
		app.replyWithError(writer, ErrAppGateMissingAppAddress)
		log.Print("ERROR: no application address provided")
	}

	// TODO_TECHDEBT: Currently, there is no information about the RPC type requested. It should
	// be extracted from the request and used to determine the RPC type to handle. handle*Relay()
	// calls should be wrapped into a switch statement to handle different types of relays.
	err := app.handleJSONRPCRelay(ctx, appAddress, serviceId, request, writer)
	if err != nil {
		// Reply with an error response if there was an error handling the relay.
		app.replyWithError(writer, err)
		log.Printf("ERROR: failed handling relay: %s", err)
		return
	}

	log.Print("INFO: request serviced successfully")
}

// replyWithError replies to the application with an error response.
// TODO_TECHDEBT: This method should be aware of the nature of the error to use the appropriate JSONRPC
// Code, Message and Data. Possibly by augmenting the passed in error with the adequate information.
func (app *appGateServer) replyWithError(writer http.ResponseWriter, err error) {
	relayResponse := &types.RelayResponse{
		Payload: &types.RelayResponse_JsonRpcPayload{
			JsonRpcPayload: &types.JSONRPCResponsePayload{
				Id:      make([]byte, 0),
				Jsonrpc: "2.0",
				Error: &types.JSONRPCResponseError{
					// Using conventional error code indicating internal server error.
					Code:    -32000,
					Message: err.Error(),
					Data:    nil,
				},
			},
		},
	}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		log.Printf("ERROR: failed marshaling relay response: %s", err)
		return
	}

	if _, err = writer.Write(relayResponseBz); err != nil {
		log.Printf("ERROR: failed writing relay response: %s", err)
		return
	}
}

// validateConfig validates the appGateServer configuration.
func (app *appGateServer) validateConfig() error {
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
