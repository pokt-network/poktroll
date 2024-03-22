package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ relayer.RelayServer = (*synchronousRPCServer)(nil)

func init() {
	reg := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(reg)
}

// synchronousRPCServer is the struct that holds the state of the synchronous
// RPC server. It is used to listen for and respond to relay requests where
// there is a one-to-one correspondence between relay requests and relay responses.
type synchronousRPCServer struct {
	logger polylog.Logger

	// supplierServiceMap is a map of serviceId -> SupplierServiceConfig
	// representing the supplier's advertised services.
	supplierServiceMap map[string]*sharedtypes.Service

	// proxyConfig is the configuration of the proxy server. It contains the
	// host address of the server, the service endpoint, and the advertised service.
	// endpoints it gets relay requests from.
	proxyConfig *config.RelayMinerProxyConfig

	// server is the HTTP server that listens for incoming relay requests.
	server *http.Server

	// relayerProxy is the main relayer proxy that the server uses to perform its operations.
	relayerProxy relayer.RelayerProxy

	// servedRelaysProducer is a channel that emits the relays that have been served, allowing
	// the servedRelays observable to fan-out notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay
}

// NewSynchronousServer creates a new HTTP server that listens for incoming
// relay requests and forwards them to the supported proxied service endpoint.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments
// and returns a RelayServer that listens to incoming RelayRequests.
func NewSynchronousServer(
	logger polylog.Logger,
	proxyConfig *config.RelayMinerProxyConfig,
	supplierServiceMap map[string]*sharedtypes.Service,
	servedRelaysProducer chan<- *types.Relay,
	proxy relayer.RelayerProxy,
) relayer.RelayServer {
	return &synchronousRPCServer{
		logger:               logger,
		supplierServiceMap:   supplierServiceMap,
		server:               &http.Server{Addr: proxyConfig.Host},
		relayerProxy:         proxy,
		servedRelaysProducer: servedRelaysProducer,
		proxyConfig:          proxyConfig,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to end before shutting down.
// This method is blocking and should be called in a goroutine.
func (sync *synchronousRPCServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		sync.server.Shutdown(ctx)
	}()

	// Set the HTTP handler.
	sync.server.Handler = sync

	return sync.server.ListenAndServe()
}

// Stop terminates the service server and returns an error if it fails.
func (sync *synchronousRPCServer) Stop(ctx context.Context) error {
	return sync.server.Shutdown(ctx)
}

// ServeHTTP listens for incoming relay requests. It implements the respective
// method of the http.Handler interface. It is called by http.ListenAndServe()
// when synchronousRPCServer is used as an http.Handler with an http.Server.
// (see https://pkg.go.dev/net/http#Handler)
func (sync *synchronousRPCServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()
	ctx := request.Context()

	var originHost string
	// When the proxy is behind a reverse proxy, or is getting its requests from
	// a CDN or a load balancer, the host header may not contain the on-chain
	// advertized address needed to determine the service that the relay request is for.
	// These CDNs and reverse proxies usually set the X-Forwarded-Host header
	// to the original host.
	// RelayMiner operators that have such a setup can set the XForwardedHostLookup
	// option to true in the proxy config to enable the proxy to look up the
	// original host from the X-Forwarded-Host header.
	// Get the original host from X-Forwarded-Host header if specified in the proxy
	// config and fall back to the Host header if it is not specified.
	if sync.proxyConfig.XForwardedHostLookup {
		originHost = request.Header.Get("X-Forwarded-Host")
	}

	if originHost == "" {
		originHost = request.Host
	}

	var supplierService *sharedtypes.Service
	var serviceUrl *url.URL

	// Get the Service and serviceUrl corresponding to the originHost.
	// TODO_IMPROVE(red-0ne): Checking that the originHost is currently done by
	// iterating over the proxy config's suppliers and checking if the originHost
	// is present in any of the supplier's service's hosts. We could improve this
	// by building a map at the server initialization level with originHost as the
	// key so that we can get the service and serviceUrl in O(1) time.
	for _, supplierServiceConfig := range sync.proxyConfig.Suppliers {
		for _, host := range supplierServiceConfig.Hosts {
			if host == originHost {
				supplierService = sync.supplierServiceMap[supplierServiceConfig.ServiceId]
				serviceUrl = supplierServiceConfig.ServiceConfig.Url
				break
			}
		}

		if serviceUrl != nil {
			break
		}
	}

	if supplierService == nil || serviceUrl == nil {
		sync.replyWithError(
			ctx,
			[]byte{},
			writer,
			sync.proxyConfig.ProxyName,
			"unknown",
			ErrRelayerProxyServiceEndpointNotHandled,
		)
		return
	}

	// Increment the relays counter.
	relaysTotal.With("proxy_name", sync.proxyConfig.ProxyName, "service_id", supplierService.Id).Add(1)
	defer func() {
		duration := time.Since(startTime).Seconds()

		// Capture the relay request duration metric.
		relaysDurationSeconds.With(
			"proxy_name", sync.proxyConfig.ProxyName,
			"service_id", supplierService.Id).Observe(duration)
	}()

	sync.logger.Debug().Msg("serving synchronous relay request")

	// Extract the relay request from the request body.
	sync.logger.Debug().Msg("extracting relay request from request body")
	relayRequest, err := sync.newRelayRequest(request)
	if err != nil {
		sync.replyWithError(ctx, []byte{}, writer, sync.proxyConfig.ProxyName, supplierService.Id, err)
		sync.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	// Relay the request to the proxied service and build the response that will be sent back to the client.
	relay, err := sync.serveHTTP(ctx, serviceUrl, supplierService, request, relayRequest)
	if err != nil {
		// Reply with an error if the relay could not be served.
		sync.replyWithError(ctx, relayRequest.Payload, writer, sync.proxyConfig.ProxyName, supplierService.Id, err)
		sync.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	// Send the relay response to the client.
	if err := sync.sendRelayResponse(relay.Res, writer); err != nil {
		sync.replyWithError(ctx, relayRequest.Payload, writer, sync.proxyConfig.ProxyName, supplierService.Id, err)
		sync.logger.Warn().Err(err).Msg("failed sending relay response")
		return
	}

	sync.logger.Info().Fields(map[string]any{
		"application_address":  relay.Res.Meta.SessionHeader.ApplicationAddress,
		"service_id":           relay.Res.Meta.SessionHeader.Service.Id,
		"session_start_height": relay.Res.Meta.SessionHeader.SessionStartBlockHeight,
		"server_addr":          sync.server.Addr,
	}).Msg("relay request served successfully")

	// Emit the relay to the servedRelays observable.
	sync.servedRelaysProducer <- relay
}

// serveHTTP holds the underlying logic of ServeHTTP.
func (sync *synchronousRPCServer) serveHTTP(
	ctx context.Context,
	serviceUrl *url.URL,
	supplierService *sharedtypes.Service,
	request *http.Request,
	relayRequest *types.RelayRequest,
) (*types.Relay, error) {
	// Verify the relay request signature and session.
	// TODO_TECHDEBT(red-0ne): Currently, the relayer proxy is responsible for verifying
	// the relay request signature. This responsibility should be shifted to the relayer itself.
	// Consider using a middleware pattern to handle non-proxy specific logic, such as
	// request signature verification, session verification, and response signature.
	// This would help in separating concerns and improving code maintainability.
	// See https://github.com/pokt-network/poktroll/issues/160
	if err := sync.relayerProxy.VerifyRelayRequest(ctx, relayRequest, supplierService); err != nil {
		return nil, err
	}

	// Get the relayRequest payload's `io.ReadCloser` to add it to the http.Request
	// that will be sent to the proxied (i.e. staked for) service.
	// (see https://pkg.go.dev/net/http#Request) Body field type.
	requestBodyReader := io.NopCloser(bytes.NewBuffer(relayRequest.Payload))
	sync.logger.Debug().
		Str("request_payload", string(relayRequest.Payload)).
		Msg("serving relay request")

	// Build the request to be sent to the native service by substituting
	// the destination URL's host with the native service's listen address.
	sync.logger.Debug().
		Str("destination_url", serviceUrl.String()).
		Msg("building relay request payload to service")

	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    serviceUrl,
		Host:   serviceUrl.Host,
		Body:   requestBodyReader,
	}

	// Send the relay request to the native service.
	httpResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, err
	}

	// Build the relay response from the native service response
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it was verified to be valid
	// and has to be the same as the relayResponse session header.
	sync.logger.Debug().
		Str("relay_request_session_header", relayRequest.Meta.SessionHeader.String()).
		Msg("building relay response protobuf from service response")

	relayResponse, err := sync.newRelayResponse(httpResponse, relayRequest.Meta.SessionHeader)
	if err != nil {
		return nil, err
	}

	return &types.Relay{Req: relayRequest, Res: relayResponse}, nil
}

// sendRelayResponse marshals the relay response and sends it to the client.
func (sync *synchronousRPCServer) sendRelayResponse(
	relayResponse *types.RelayResponse,
	writer http.ResponseWriter,
) error {
	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		return err
	}

	_, err = writer.Write(relayResponseBz)
	return err
}
