package proxy

import (
	"context"
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

	// servedRelaysProducer is a channel that emits the relays that have been served, allowing
	// the servedRelays observable to fan-out notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay

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
		// TODO_IMPROVE: Make timeouts configurable.
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &relayMinerHTTPServer{
		logger:               logger,
		server:               httpServer,
		relayAuthenticator:   relayAuthenticator,
		servedRelaysProducer: servedRelaysProducer,
		serverConfig:         serverConfig,
		relayMeter:           relayMeter,
		blockClient:          blockClient,
		sharedQueryClient:    sharedQueryClient,
		sessionQueryClient:   sessionQueryClient,
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

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when relayMinerHTTPServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (server *relayMinerHTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	// Determine whether the request is upgrading to websocket.
	if isWebSocketRequest(request) {
		server.logger.Debug().Msg("detected asynchronous relay request")

		if err := server.handleAsyncConnection(ctx, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, nil, writer)
			server.logger.Warn().Err(err).Msg("failed serving asynchronous relay request")
			return
		}
	} else {
		server.logger.Debug().Msg("detected synchronous relay request")

		if relayRequest, err := server.serveSyncRequest(ctx, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, relayRequest, writer)
			server.logger.Warn().Err(err).Msg("failed serving synchronous relay request")
			return
		}
	}
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
