package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	validate "github.com/go-playground/validator/v10"

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
			return err
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= http.StatusInternalServerError {
			return errors.New("ping failed")
		}

	}

	return nil
}

// forwardPayload represents the request body format to forward a request to
// the supplier.
type forwardPayload struct {
	Method  string            `json:"method" validate:"required,oneof=GET PATCH PUT CONNECT TRACE DELETE POST HEAD OPTIONS"`
	Path    string            `json:"path" validate:"required"`
	Headers map[string]string `json:"headers"`
	Data    string            `json:"data"`
}

// toHeaders instantiates an http.Header based on the Headers field.
func (p forwardPayload) toHeaders() http.Header {
	h := http.Header{}

	for k, v := range p.Headers {
		h.Set(k, v)
	}

	return h
}

// Validate returns true if the payload format is correct.
func (p forwardPayload) Validate() error {
	var err error
	if structErr := validate.New().Struct(&p); structErr != nil {
		for _, e := range structErr.(validate.ValidationErrors) {
			err = errors.Join(err, e)
		}
	}

	return err
}

// Forward reads the forward payload request and sends a request to a managed service id.
func (server *relayMinerHTTPServer) Forward(ctx context.Context, serviceID string, w http.ResponseWriter, req *http.Request) error {
	supplierConfig, ok := server.serverConfig.SupplierConfigsMap[serviceID]
	if !ok {
		return ErrRelayerProxyServiceIDNotFound
	}

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	var payload forwardPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}

	if err := payload.Validate(); err != nil {
		return err
	}

	url := *supplierConfig.ServiceConfig.BackendUrl
	url.Path = path.Join(url.Path, payload.Path)

	forwardReq := &http.Request{
		Method: payload.Method,
		Body:   io.NopCloser(bytes.NewBufferString(payload.Data)),
		URL:    &url,
		Header: payload.toHeaders(),
	}

	c := http.Client{
		Transport: http.DefaultTransport,
	}

	// forward request to the supplier.
	resp, err := c.Do(forwardReq)
	if err != nil {
		server.logger.Error().Fields(map[string]any{
			"service_id": serviceID,
			"method":     payload.Method,
			"path":       payload.Path,
			"headers":    payload.Headers,
		}).Err(err).Msg("failed to send forward http request")

		if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
			http.Error(w, fmt.Sprintf("relayminer: foward http request timeout exceeded"), http.StatusGatewayTimeout)
		} else {
			http.Error(w, fmt.Sprintf("relayminer: error forward http request: %s", err.Error()), http.StatusInternalServerError)
		}
		return err
	}

	w.WriteHeader(resp.StatusCode)

	// streaming supplier's output to the client.
	if _, err := io.Copy(w, resp.Body); err != nil {
		server.logger.Error().Fields(map[string]any{
			"service_id": serviceID,
			"method":     payload.Method,
			"path":       payload.Path,
			"headers":    payload.Headers,
		}).Err(err).Msg("failed to write forward http reponse")

		http.Error(w, fmt.Sprintf("relayminer: error on forward http response: %s", err.Error()), http.StatusInternalServerError)

		return err
	}

	return nil
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
