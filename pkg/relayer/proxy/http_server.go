package proxy

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// rpcTypeHeader is the header key for the RPC type, provided by the client.
const RPCTypeHeader = "Rpc-Type"

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

	// knownSession is a map of known session IDs to their corresponding session end block heights.
	// It is used to cache session information to avoid redundant validations and queries.
	// The map is protected by a RWMutex to allow concurrent access.
	knownSessions      map[string]int64
	knownSessionsMutex *sync.RWMutex

	// eagerValidationEnabled indicates whether eager validation is enabled.
	// When enabled, all incoming relay requests are validated immediately upon receipt.
	// When disabled, relay requests are:
	// 1. Validated immediately if their session is known
	// 2. Deferred for validation if their session is unknown
	eagerValidationEnabled bool

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
	// Create the HTTP server with comprehensive limits for security and stability.
	httpServer := &http.Server{
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  config.DefaultRequestTimeoutDuration,
		WriteTimeout: config.DefaultRequestTimeoutDuration,
		// MaxHeaderBytes limits header size to prevent memory exhaustion (1MB limit)
		MaxHeaderBytes: 1 << 20, // 1MB

		// ConnState tracks connection lifecycle for debugging "missing supplier operator signature" errors
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateClosed:
				logger.Debug().Str("remote_addr", conn.RemoteAddr().String()).Msg("HTTP connection closed")
			case http.StateHijacked:
				logger.Debug().Str("remote_addr", conn.RemoteAddr().String()).Msg("HTTP connection hijacked")
			}
		},
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
		knownSessions:                  make(map[string]int64),
		knownSessionsMutex:             &sync.RWMutex{},
		eagerValidationEnabled:         serverConfig.EnableEagerValidation,
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

	// Subscribe to new blocks to prune outdated known sessions.
	committedBlocksSequence := server.blockClient.CommittedBlocksSequence(ctx)
	channel.ForEach(ctx, committedBlocksSequence, server.pruneOutdatedKnownSessions)

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

	// Create a request-specific logger to avoid concurrent access issues
	logger := server.logger.With(
		"request_id", request.Header.Get("X-Request-ID"),
		"user_agent", request.Header.Get("User-Agent"),
		"remote_addr", request.RemoteAddr,
	)

	// isWebSocketRequest checks if the request is trying to upgrade to WebSocket.
	isWebSocketRequest := func(r *http.Request) bool {
		// The request must have the "Rpc-Type" header set to "websocket".
		// This will be handled in the client, likely a PATH gateway.
		return r.Header.Get(RPCTypeHeader) == strconv.Itoa(int(sharedtypes.RPCType_WEBSOCKET))
	}

	// Determine whether the request is upgrading to websocket.
	if isWebSocketRequest(request) {
		logger.ProbabilisticDebugInfo(relayProbabilisticDebugProb).Msg("🔍 detected asynchronous relay request")

		if err := server.handleAsyncConnection(ctx, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, nil, writer)
			logger.Warn().Err(err).Msg("❌ failed serving asynchronous relay request")
			return
		}
	} else {
		logger.ProbabilisticDebugInfo(relayProbabilisticDebugProb).Msg("🔍 detected synchronous relay request")

		if relayRequest, err := server.serveSyncRequest(ctx, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, relayRequest, writer)

			// Do not alarm the RelayMiner operator if the error is a client error
			if ErrRelayerProxyInternalError.Is(err) {
				logger.Error().Err(err).Msgf("❌ Failed serving synchronous relay request. This COULD be a configuration issue on the RelayMiner! Please check your setup. ⚙️🛠️")
			} else {
				logger.Error().Err(err).Msgf("⚠️ Failed serving synchronous relay request. This MIGHT be a client error.")
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
	timeout := time.Duration(config.DefaultRequestTimeoutSeconds) * time.Second

	// Look up service-specific timeout in server config
	if supplierConfig, exists := server.serverConfig.SupplierConfigsMap[serviceId]; exists {
		timeout = time.Duration(supplierConfig.RequestTimeoutSeconds) * time.Second
	}

	return timeout
}
