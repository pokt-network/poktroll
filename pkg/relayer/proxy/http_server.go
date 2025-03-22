package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

// - relayProbabilisticDebugProb is the probability of a debug log being shown for a relay request.
// - This has to be very low to avoid spamming the logs for RelayMiners that end up serving millions of relays.
// - In the case of errors, it increases the likelihood of seeing issues in the logs
var relayProbabilisticDebugProb float64 = 0.0001

var _ relayer.RelayServer = (*relayMinerHTTPServer)(nil)

func init() {
	reg := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(reg)
}

// relayMinerHTTPServer is the struct that holds the state of the RelayMiner's HTTP server.
// It accepts incoming relay requests coming from the Gateway and forwards them to the
// corresponding service endpoint.
// It supports both synchronous (e.g. request/response) as well as asynchronous
// (e.g. websocket) relay requests.
// DEV_NOTE: The relayMinerHTTPServer:
//   - Serves as a communication bridge between the Gateway and the RelayMiner.
//   - It processes ALL incoming relay requests regardless of their the RPC type
//     (e.g. JSON_RPC, REST, gRPC, Websockets...).
type relayMinerHTTPServer struct {
	logger polylog.Logger

	// serverConfig is the RelayMiner's proxy server configuration.
	// It contains the host address of the server, the service endpoint, and the
	// advertised service endpoints it gets relay requests from.
	serverConfig *config.RelayMinerServerConfig

	// server is the HTTP server that listens for incoming relay requests.
	server *http.Server

	// relayAuthenticator is the RelayMiner's relay authenticator that validates
	// the relay requests and signs the relay responses.
	relayAuthenticator relayer.RelayAuthenticator

	// servedRewardableRelaysProducer is a channel that emits the relays thatL
	// 	1. Have been successfully served
	// 	2. Are reward-applicable (i.e. should be inserted into the SMT)
	// Some examples of relays that shouldn't be emitted to this channel:
	// 	- Relays that failed to be served
	// 	- Relays that are not reward-applicable
	// 	- Relays that are over-serviced
	// The servedRewardableRelaysProducer observable to fan-out notifications to its subscribers.
	servedRewardableRelaysProducer chan<- *types.Relay

	// relayMeter is the relay meter that the RelayServer uses to meter the relays and claim the relay price.
	// It is used to ensure that the relays are metered and priced correctly.
	relayMeter relayer.RelayMeter

	// Query clients used to query for the served session's parameters.
	blockClient        client.BlockClient
	sharedQueryClient  client.SharedQueryClient
	sessionQueryClient client.SessionQueryClient
}

// NewHTTPServer creates a new RelayServer that listens for incoming relay requests
// and forwards them to the corresponding proxied service endpoint.
// TODO_RESEARCH(#590): Currently, the communication between the Gateway and the
// RelayMiner uses HTTP. This could be changed to a more generic and performant
// one, such as QUIC or pure TCP.
func NewHTTPServer(
	logger polylog.Logger,
	serverConfig *config.RelayMinerServerConfig,
	servedRelaysProducer chan<- *types.Relay,
	relayAuthenticator relayer.RelayAuthenticator,
	relayMeter relayer.RelayMeter,
	blockClient client.BlockClient,
	sharedQueryClient client.SharedQueryClient,
	sessionQueryClient client.SessionQueryClient,
) relayer.RelayServer {
	// Create the HTTP server.
	httpServer := &http.Server{
		// Keep IdleTimeout reasonable to clean up idle connections
		IdleTimeout: 60 * time.Second,
		// Read and Write timeouts are set to reasonable default values to prevent slow-loris
		// attacks and to ensure that the server does not hang indefinitely on a request.
		// These defaults are kept as baseline security measures, but per-request timeouts
		// will override these values based on the configured timeout for each service ID.
		ReadTimeout:  config.DefaultRequestTimeoutSeconds * time.Second,
		WriteTimeout: config.DefaultRequestTimeoutSeconds * time.Second,
	}

	return &relayMinerHTTPServer{
		logger:                         logger,
		server:                         httpServer,
		relayAuthenticator:             relayAuthenticator,
		servedRewardableRelaysProducer: servedRelaysProducer,
		serverConfig:                   serverConfig,
		relayMeter:                     relayMeter,
		blockClient:                    blockClient,
		sharedQueryClient:              sharedQueryClient,
		sessionQueryClient:             sessionQueryClient,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to end before shutting down.
// This method is blocking and should be called in a goroutine.
func (server *relayMinerHTTPServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = server.server.Shutdown(ctx)
	}()

	// Set the HTTP handler.
	server.server.Handler = server

	listener, err := net.Listen("tcp", server.serverConfig.ListenAddress)
	if err != nil {
		server.logger.Error().Err(err).Msg("failed to create listener")
		return err
	}

	return server.server.Serve(listener)
}

// Stop terminates the service server and returns an error if it fails.
func (server *relayMinerHTTPServer) Stop(ctx context.Context) error {
	return server.server.Shutdown(ctx)
}

// Ping tries to dial the suppliers backend URLs to test the connection.
func (server *relayMinerHTTPServer) Ping(ctx context.Context) error {
	for _, supplierCfg := range server.serverConfig.SupplierConfigsMap {
		c := &http.Client{Timeout: 2 * time.Second}

		backendUrl := *supplierCfg.ServiceConfig.BackendUrl
		if backendUrl.Scheme == "ws" || backendUrl.Scheme == "wss" {
			// TODO_IMPROVE: Consider testing websocket connectivity by establishing
			// a websocket connection instead of using an HTTP connection.
			server.logger.Warn().Msgf(
				"backend URL %s scheme is a %s, switching to http to check connectivity",
				backendUrl.String(),
				backendUrl.Scheme,
			)

			if backendUrl.Scheme == "ws" {
				backendUrl.Scheme = "http"
			} else {
				backendUrl.Scheme = "https"
			}
		}
		resp, err := c.Head(backendUrl.String())
		if err != nil {
			return fmt.Errorf(
				"failed to ping backend %q for serviceId %q: %w",
				backendUrl.String(), supplierCfg.ServiceId, err,
			)
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf(
				"failed to ping backend %q for serviceId %q: received status code %d",
				backendUrl.String(), supplierCfg.ServiceId, resp.StatusCode,
			)
		}

	}

	return nil
}

// Forward reads the forward payload request and sends a request to a managed service id.
func (server *relayMinerHTTPServer) Forward(ctx context.Context, serviceID string, w http.ResponseWriter, req *http.Request) error {
	supplierConfig, ok := server.serverConfig.SupplierConfigsMap[serviceID]
	if !ok {
		return ErrRelayerProxyServiceIDNotFound
	}

	if isWebSocketRequest(req) {
		return server.forwardAsyncConnection(ctx, supplierConfig, w, req)
	} else {
		return server.forwardHTTP(ctx, supplierConfig, w, req)
	}
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when relayMinerHTTPServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (server *relayMinerHTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	// Determine whether the request is upgrading to websocket.
	if isWebSocketRequest(request) {
		server.logger.ProbabilisticDebugInfo(relayProbabilisticDebugProb).Msg("🔍 detected asynchronous relay request")

		if err := server.handleAsyncConnection(ctx, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, nil, writer)
			server.logger.Warn().Err(err).Msg("❌ failed serving asynchronous relay request")
			return
		}
	} else {
		server.logger.ProbabilisticDebugInfo(relayProbabilisticDebugProb).Msg("🔍 detected synchronous relay request")

		if relayRequest, err := server.serveSyncRequest(ctx, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, relayRequest, writer)

			// Do not alarm the RelayMiner operator if the error is a client error
			if ErrRelayerProxyInternalError.Is(err) {
				server.logger.Error().Err(err).Msgf("❌ Failed serving synchronous relay request. This COULD be a configuration issue on the RelayMiner! Please check your setup. ⚙️🛠️")
			} else {
				server.logger.Error().Err(err).Msgf("⚠️ Failed serving synchronous relay request. This MIGHT be a client error.")
			}
			return
		}
	}
}

// requestTimeoutForServiceId determines the timeout for the relay request
// based on the service ID.
//   - It looks up the service ID in the server's configuration and returns the
//     timeout specified for that service ID.
//   - If no specific timeout is found, it returns the default timeout.
func (server *relayMinerHTTPServer) requestTimeoutForServiceId(serviceId string) time.Duration {
	timeout := config.DefaultRequestTimeoutSeconds * time.Second

	// Look up service-specific timeout in server config
	if supplierConfig, exists := server.serverConfig.SupplierConfigsMap[serviceId]; exists {
		timeout = time.Duration(supplierConfig.RequestTimeoutSeconds) * time.Second
	}

	return timeout
}

// isWebSocketRequest checks if the request is trying to upgrade to WebSocket.
func isWebSocketRequest(r *http.Request) bool {
	// Check if the request is trying to upgrade to WebSocket as per the RFC 6455.
	// The request must have the "Upgrade" and "Connection" headers set to
	// "websocket" and "Upgrade" respectively.
	// refer to: https://datatracker.ietf.org/doc/html/rfc6455#section-4.2.1
	upgradeHeader := r.Header.Get("Upgrade")
	connectionHeader := r.Header.Get("Connection")

	return http.CanonicalHeaderKey(upgradeHeader) == "Websocket" &&
		http.CanonicalHeaderKey(connectionHeader) == "Upgrade"
}
