package appgateserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metricsmiddleware "github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"

	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	"github.com/pokt-network/poktroll/pkg/partials"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/sdk"
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

	// clientCtx is the client context for the application.
	// It is used to query for the application's account to unmarshal the supplier's account
	// and get the public key to verify the relay response signature.
	clientCtx querytypes.Context

	// sdk is the POKTRollSDK that the appGateServer uses to query for the current session
	// and send relay requests to the supplier.
	sdk sdk.POKTRollSDK

	// listeningEndpoint is the endpoint that the appGateServer will listen on.
	listeningEndpoint *url.URL

	// server is the HTTP server that will be used capture application requests
	// so that they can be signed and relayed to the supplier.
	server *http.Server

	// endpointSelectionIndexMu is a mutex that protects the endpointSelectionIndex
	// from concurrent relay requests.
	endpointSelectionIndexMu sync.Mutex

	// endpointSelectionIndex is the index of the last selected endpoint.
	// It is used to cycle through the available endpoints in a round-robin fashion.
	endpointSelectionIndex int
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
	app := &appGateServer{}

	if err := depinject.Inject(
		deps,
		&app.logger,
		&app.clientCtx,
		&app.sdk,
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

	// TODO_CONSIDERATION: Use app.listeningEndpoint scheme to determine which
	// kind of server to create (HTTP, HTTPS, TCP, UNIX, etc...)
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

	// This hooks https://github.com/slok/go-http-metrics to the appgate server HTTP server.
	mm := metricsmiddleware.New(metricsmiddleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})

	// Set the HTTP handler.
	app.server.Handler = middlewarestd.Handler("", mm, app)

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
//	"<protocol>://host:port/serviceId[/other/path/segments]?applicationAddr=<applicationAddr>"
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
	requestPayloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		app.replyWithError(
			ctx,
			requestPayloadBz,
			writer,
			serviceId,
			"unknown",
			ErrAppGateHandleRelay.Wrapf("reading relay request body: %s", err),
		)
		// TODO_IMPROVE: log additional info?
		app.logger.Error().Err(err).Msg("failed reading relay request body")

		return
	}
	app.logger.Debug().
		Str("service_id", serviceId).
		Str("payload", string(requestPayloadBz)).
		Msg("handling relay")

	// TODO_IMPROVE: log additional info?
	app.logger.Debug().Msg("determining request type")

	// Get the type of the request by doing a partial unmarshal of the payload
	requestType, err := partials.GetRequestType(ctx, requestPayloadBz)
	if err != nil {
		app.replyWithError(ctx, requestPayloadBz, writer, serviceId, "unknown", ErrAppGateHandleRelay)
		// TODO_IMPROVE: log additional info?
		app.logger.Error().Err(err).Msg("failed getting request type")

		return
	}

	// Determine the application address.
	appAddress := app.signingInformation.AppAddress
	if appAddress == "" {
		appAddress = request.URL.Query().Get("applicationAddr")
	}
	if appAddress == "" {
		app.replyWithError(ctx, requestPayloadBz, writer, serviceId, requestType.String(), ErrAppGateMissingAppAddress)
		// TODO_IMPROVE: log additional info?
		app.logger.Error().Msg("no application address provided")

		return
	}

	// Put the request body bytes back into the request body.
	request.Body = io.NopCloser(bytes.NewBuffer(requestPayloadBz))

	// TODO(@h5law, @red0ne): Add support for asynchronous relays, and switch on
	// the request type here.
	// TODO_RESEARCH: Should this be started in a goroutine, to allow for
	// concurrent requests from numerous applications?
	if err := app.handleSynchronousRelay(
		ctx, appAddress, serviceId, requestPayloadBz, requestType, request, writer); err != nil {

		// Reply with an error response if there was an error handling the relay.
		app.replyWithError(ctx, requestPayloadBz, writer, serviceId, requestType.String(), err)
		// TODO_IMPROVE: log additional info?
		app.logger.Error().Err(err).Msg("failed handling relay")

		return
	}

	// TODO_IMPROVE: log additional info?
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

// Starts a metrics server on the given address.
func (app *appGateServer) ServeMetrics(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		app.logger.Error().Err(err).Msg("failed to listen on address for metrics")
		return err
	}

	// If no error, start the server in a new goroutine
	go func() {
		app.logger.Info().Str("endpoint", addr).Msg("serving metrics")
		if err := http.Serve(ln, promhttp.Handler()); err != nil {
			app.logger.Error().Err(err).Msg("metrics server failed")
			return
		}
	}()

	return nil
}

type appGateServerOption func(*appGateServer)
