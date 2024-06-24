package appgateserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"strings"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/shannon-sdk/sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	metricsmiddleware "github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"

	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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

	// sdk is the ShannonSDK that the appGateServer uses to query for the current session
	// and send relay requests to the supplier.
	sdk *sdk.ShannonSDK

	// listeningEndpoint is the endpoint that the appGateServer will listen on.
	listeningEndpoint *url.URL

	// server is the HTTP server that will be used capture application requests
	// so that they can be signed and relayed to the supplier.
	server *http.Server

	// endpointSelectionIndexMu is a mutex that protects the endpointSelectionIndex
	// from concurrent relay requests.
	endpointSelectionIndexMu sync.Mutex

	// endpointSelectionIndex is the index of the last selected endpoint.
	// It is used to cycle through the available endpoints using a round-robin strategy.
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
	// TODO_RESEARCH(#590): Currently, the communication between the AppGateServer and the
	// RelayMiner uses HTTP. This could be changed to a more generic and performant
	// one, such as pure TCP.
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

	newUrlPath, serviceId := extractServiceId(request.URL.Path)
	request.URL.Path = newUrlPath

	// Create an error logger with the common fields for error logging.
	errorLogger := app.logger.With().
		Error().
		Str("service_id", serviceId).
		Fields(map[string]interface{}{
			"method":         request.Method,
			"url":            request.URL.String(),
			"content_type":   request.Header.Get("Content-Type"),
			"content_length": request.ContentLength,
		})

	logger := app.logger.With().Debug().Str("service_id", serviceId)

	poktHTTPRequest, requestBz, err := sdktypes.SerializeHTTPRequest(request)
	if err != nil {
		// If the request cannot not be parsed, pass an empty POKTHTTPRequest and
		// an UNKNOWN_RPC type to the replyWithError method.
		emptyPOKTHTTPRequest := &sdktypes.POKTHTTPRequest{}
		rpcType := sharedtypes.RPCType_UNKNOWN_RPC
		errorReply := ErrAppGateHandleRelay.Wrapf("parsing request: %s", err)

		app.replyWithError(errorReply, emptyPOKTHTTPRequest, serviceId, rpcType, writer)
		errorLogger.Err(err).Msg("failed parsing request")

		return
	}

	logger.Msg("handling relay")

	// Get the type of the request by inspecting the request properties.
	rpcType := poktHTTPRequest.GetRPCType()

	// Add newly available fields to the error logger.
	errorLogger = errorLogger.
		Str("rpc_type", rpcType.String()).
		Str("payload", string(poktHTTPRequest.BodyBz))

	logger = logger.Str("rpc_type", rpcType.String())
	logger.Msg("identified rpc type")

	// Determine the application address.
	appAddress := app.signingInformation.AppAddress
	if appAddress == "" {
		appAddress = request.URL.Query().Get("applicationAddr")
	}
	if appAddress == "" {
		// If no application address is provided, reply with an error response.
		app.replyWithError(ErrAppGateMissingAppAddress, poktHTTPRequest, serviceId, rpcType, writer)
		errorLogger.Err(ErrAppGateMissingAppAddress).Msg("no application address provided")

		return
	}

	logger = logger.Str("application_addr", appAddress)
	errorLogger = errorLogger.Str("application_addr", appAddress)

	// Create a requestInfo struct to pass to the handleSynchronousRelay method.
	reqInfo := &requestInfo{
		appAddress:  appAddress,
		serviceId:   serviceId,
		rpcType:     rpcType,
		poktRequest: poktHTTPRequest,
		requestBz:   requestBz,
	}

	// TODO_IMPROVE(@red-0ne, #40): Add support for asynchronous relays, and switch on
	// the request type here.
	if err := app.handleSynchronousRelay(ctx, reqInfo, writer); err != nil {
		// Reply with an error response if there was an error handling the relay.
		app.replyWithError(err, poktHTTPRequest, serviceId, rpcType, writer)
		errorLogger.Err(err).Msg("failed handling relay")

		return
	}

	logger.Msg("request serviced successfully")
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

// Starts a pprof server on the given address.
func (app *appGateServer) ServePprof(ctx context.Context, addr string) error {
	pprofMux := http.NewServeMux()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:    addr,
		Handler: pprofMux,
	}

	// If no error, start the server in a new goroutine
	go func() {
		app.logger.Info().Str("endpoint", addr).Msg("starting a pprof endpoint")
		server.ListenAndServe()
	}()

	go func() {
		<-ctx.Done()
		app.logger.Info().Str("endpoint", addr).Msg("stopping a pprof endpoint")
		server.Shutdown(ctx)
	}()

	return nil
}

// extractServiceId extracts the serviceId from the request path and returns it
// along with the new request path that is stripped of the serviceId.
func extractServiceId(urlPath string) (newUrlPath string, serviceId string) {
	// Extract the serviceId from the request path.
	serviceId = strings.Split(urlPath, "/")[1]

	// Remove the serviceId from the request path which is specific AppGateServer business logic.
	// The remaining path is the path of the request that will be serialized and
	// sent to sent to the supplier within a RelayRequest.
	// For example:
	// * Assume a request to the Backend service has to be made with the path "/backend/relay"
	// * The AppGateServer expects the request from the client to have the path
	//   "/serviceId/backend/relay"
	// * The AppGateServer will remove the serviceId from the path, serialize the request
	// and send it to the supplier with the path "/backend/relay"
	//
	// This is specific logic to how the AppGateServer functions. Other gateways
	// may have different means or approaches of identifying the service that the
	// request is for (e.g. POST data).
	newUrlPath = strings.TrimPrefix(urlPath, fmt.Sprintf("/%s", serviceId))
	if newUrlPath == "" {
		newUrlPath = "/"
	}

	return newUrlPath, serviceId
}

type appGateServerOption func(*appGateServer)
