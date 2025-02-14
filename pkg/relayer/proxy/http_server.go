package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

var _ relayer.RelayServer = (*httpServer)(nil)

func init() {
	reg := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(reg)
}

// httpServer is the struct that holds the state of the synchronous
// RPC server. It is used to listen for and respond to relay requests where
// there is a one-to-one correspondence between relay requests and relay responses.
type httpServer struct {
	logger polylog.Logger

	// serverConfig is the configuration of the proxy server. It contains the
	// host address of the server, the service endpoint, and the advertised service.
	// endpoints it gets relay requests from.
	serverConfig *config.RelayMinerServerConfig

	// server is the HTTP server that listens for incoming relay requests.
	server *http.Server

	// relayAuthenticator is the main relayer proxy that the server uses to perform its operations.
	relayAuthenticator relayer.RelayAuthenticator

	// servedRelaysProducer is a channel that emits the relays that have been served, allowing
	// the servedRelays observable to fan-out notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay

	// relayMeter is the relay meter that the RelayServer uses to meter the relays and claim the relay price.
	// It is used to ensure that the relays are metered and priced correctly.
	relayMeter relayer.RelayMeter

	blockClient       client.BlockClient
	sharedQueryClient client.SharedQueryClient
}

// NewHTTPServer creates a new HTTP server that listens for incoming
// relay requests and forwards them to the supported proxied service endpoint.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments
// and returns a RelayServer that listens to incoming RelayRequests.
// TODO_RESEARCH(#590): Currently, the communication between the Gateway and the
// RelayMiner uses HTTP. This could be changed to a more generic and performant
// one, such as pure TCP.
func NewHTTPServer(
	logger polylog.Logger,
	serverConfig *config.RelayMinerServerConfig,
	servedRelaysProducer chan<- *types.Relay,
	relayAuthenticator relayer.RelayAuthenticator,
	relayMeter relayer.RelayMeter,
	blockClient client.BlockClient,
	sharedQueryClient client.SharedQueryClient,
) relayer.RelayServer {
	return &httpServer{
		logger:               logger,
		server:               &http.Server{Addr: serverConfig.ListenAddress},
		relayAuthenticator:   relayAuthenticator,
		servedRelaysProducer: servedRelaysProducer,
		serverConfig:         serverConfig,
		relayMeter:           relayMeter,
		blockClient:          blockClient,
		sharedQueryClient:    sharedQueryClient,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to end before shutting down.
// This method is blocking and should be called in a goroutine.
func (server *httpServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = server.server.Shutdown(ctx)
	}()

	// Set the HTTP handler.
	server.server.Handler = server

	return server.server.ListenAndServe()
}

// Stop terminates the service server and returns an error if it fails.
func (server *httpServer) Stop(ctx context.Context) error {
	return server.server.Shutdown(ctx)
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when synchronousRPCServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (server *httpServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	server.logger.Debug().Msg("serving synchronous relay request")

	// Extract the relay request from the request body.
	server.logger.Debug().Msg("extracting relay request from request body")
	relayRequest, err := server.newRelayRequest(request)
	request.Body.Close()
	if err != nil {
		server.replyWithError(err, nil, writer)
		server.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	if err = relayRequest.ValidateBasic(); err != nil {
		server.replyWithError(err, relayRequest, writer)
		server.logger.Warn().Err(err).Msg("failed validating relay response")
		return
	}

	supplierServiceId := relayRequest.Meta.SessionHeader.ServiceId

	originHost := request.Host
	// When the proxy is behind a reverse proxy, or is getting its requests from
	// a CDN or a load balancer, the host header may not contain the onchain
	// advertized address needed to determine the service that the relay request is for.
	// These CDNs and reverse proxies usually set the X-Forwarded-Host header
	// to the original host.
	// RelayMiner operators that have such a setup can set the XForwardedHostLookup
	// option to true in the server config to enable the proxy to look up the
	// original host from the X-Forwarded-Host header.
	// Get the original host from X-Forwarded-Host header if specified in the supplier
	// config and fall back to the Host header if it is not specified.
	if server.serverConfig.XForwardedHostLookup {
		originHost = request.Header.Get("X-Forwarded-Host")
	}

	// Extract the hostname from the request's Host header to match it with the
	// publicly exposed endpoints of the supplier service which are hostnames
	// (i.e. hosts without the port number).
	// Add the http scheme to the originHost to parse it as a URL.
	originHostUrl, err := url.Parse(fmt.Sprintf("http://%s", originHost))
	if err != nil {
		// If the originHost cannot be parsed, reply with an internal error so that
		// the original error is not exposed to the client.
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		server.replyWithError(clientError, relayRequest, writer)
		return
	}

	var serviceConfig *config.RelayMinerSupplierServiceConfig

	// Get the Service and serviceUrl corresponding to the originHost.
	// TODO_IMPROVE(red-0ne): Checking that the originHost is currently done by
	// iterating over the server config's suppliers and checking if the originHost
	// is present in any of the supplier's service's hosts. We could improve this
	// by building a map at the server initialization level with originHost as the
	// key so that we can get the service and serviceUrl in O(1) time.
	for _, supplierServiceConfig := range server.serverConfig.SupplierConfigsMap {
		for _, host := range supplierServiceConfig.PubliclyExposedEndpoints {
			if host == originHostUrl.Hostname() && supplierServiceId == supplierServiceConfig.ServiceId {
				serviceConfig = supplierServiceConfig.ServiceConfig
				break
			}
		}

		if serviceConfig != nil {
			break
		}
	}

	if serviceConfig == nil {
		server.replyWithError(ErrRelayerProxyServiceEndpointNotHandled, relayRequest, writer)
		return
	}

	// Determine whether the request is upgrading to websocket.
	if isWebSocketRequest(request) {
		sharedParams, err := server.sharedQueryClient.GetParams(ctx)
		if err != nil {
			server.replyWithError(err, relayRequest, writer)
			server.logger.Warn().Err(err).Msg("failed serving relay request")
			return
		}

		if err := server.handleAsyncConnection(ctx, server.relayAuthenticator, serviceConfig, sharedParams, relayRequest, writer, request); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, relayRequest, writer)
			server.logger.Warn().Err(err).Msg("failed serving asynchronous relay request")
			return
		}
	} else {
		if err := server.serveSyncRequest(ctx, serviceConfig, supplierServiceId, relayRequest, writer); err != nil {
			// Reply with an error if the relay could not be served.
			server.replyWithError(err, relayRequest, writer)
			server.logger.Warn().Err(err).Msg("failed serving synchronous relay request")
			return
		}
	}
}

func isWebSocketRequest(r *http.Request) bool {
	// Check if the request is trying to upgrade to WebSocket
	upgrade := r.Header.Get("Upgrade")
	connection := r.Header.Get("Connection")
	secWebSocketKey := r.Header.Get("Sec-WebSocket-Key")

	return strings.ToLower(upgrade) == "websocket" &&
		strings.Contains(strings.ToLower(connection), "upgrade") &&
		secWebSocketKey != ""
}
