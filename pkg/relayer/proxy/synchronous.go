package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
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

	// serverConfig is the configuration of the proxy server. It contains the
	// host address of the server, the service endpoint, and the advertised service.
	// endpoints it gets relay requests from.
	serverConfig *config.RelayMinerServerConfig

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
	serverConfig *config.RelayMinerServerConfig,
	supplierServiceMap map[string]*sharedtypes.Service,
	servedRelaysProducer chan<- *types.Relay,
	proxy relayer.RelayerProxy,
) relayer.RelayServer {
	return &synchronousRPCServer{
		logger:               logger,
		supplierServiceMap:   supplierServiceMap,
		server:               &http.Server{Addr: serverConfig.ListenAddress},
		relayerProxy:         proxy,
		servedRelaysProducer: servedRelaysProducer,
		serverConfig:         serverConfig,
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
	defer request.Body.Close()

	sync.logger.Debug().Msg("serving synchronous relay request")
	listenAddress := sync.serverConfig.ListenAddress

	// Extract the relay request from the request body.
	sync.logger.Debug().Msg("extracting relay request from request body")
	relayRequest, err := sync.newRelayRequest(request)
	if err != nil {
		sync.replyWithError(ctx, []byte{}, writer, listenAddress, "", err)
		sync.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	if err := relayRequest.ValidateBasic(); err != nil {
		sync.replyWithError(ctx, relayRequest.Payload, writer, listenAddress, "", err)
		sync.logger.Warn().Err(err).Msg("failed validating relay response")
		return
	}

	supplierService := relayRequest.Meta.SessionHeader.Service
	requestPayload := relayRequest.Payload

	originHost := request.Host
	// When the proxy is behind a reverse proxy, or is getting its requests from
	// a CDN or a load balancer, the host header may not contain the on-chain
	// advertized address needed to determine the service that the relay request is for.
	// These CDNs and reverse proxies usually set the X-Forwarded-Host header
	// to the original host.
	// RelayMiner operators that have such a setup can set the XForwardedHostLookup
	// option to true in the server config to enable the proxy to look up the
	// original host from the X-Forwarded-Host header.
	// Get the original host from X-Forwarded-Host header if specified in the supplier
	// config and fall back to the Host header if it is not specified.
	if sync.serverConfig.XForwardedHostLookup {
		originHost = request.Header.Get("X-Forwarded-Host")
	}

	// Extract the hostname from the request's Host header to match it with the
	// publicly exposed endpoints of the supplier service which are hostnames
	// (i.e. hosts without the port number).
	// Add the http scheme to the originHost to parse it as a URL.
	originHostUrl, err := url.Parse(fmt.Sprintf("http://%s", originHost))
	if err != nil {
		sync.replyWithError(ctx, requestPayload, writer, listenAddress, supplierService.Id, err)
		return
	}

	var serviceConfig *config.RelayMinerSupplierServiceConfig

	// Get the Service and serviceUrl corresponding to the originHost.
	// TODO_IMPROVE(red-0ne): Checking that the originHost is currently done by
	// iterating over the server config's suppliers and checking if the originHost
	// is present in any of the supplier's service's hosts. We could improve this
	// by building a map at the server initialization level with originHost as the
	// key so that we can get the service and serviceUrl in O(1) time.
	for _, supplierServiceConfig := range sync.serverConfig.SupplierConfigsMap {
		for _, host := range supplierServiceConfig.PubliclyExposedEndpoints {
			if host == originHostUrl.Hostname() && supplierService.Id == supplierServiceConfig.ServiceId {
				serviceConfig = supplierServiceConfig.ServiceConfig
				break
			}
		}

		if serviceConfig != nil {
			break
		}
	}

	if serviceConfig == nil {
		sync.replyWithError(
			ctx,
			requestPayload,
			writer,
			listenAddress,
			supplierService.Id,
			ErrRelayerProxyServiceEndpointNotHandled,
		)
		return
	}

	// Increment the relays counter.
	relaysTotal.With("service_id", supplierService.Id).Add(1)
	defer func() {
		duration := time.Since(startTime).Seconds()

		// Capture the relay request duration metric.
		relaysDurationSeconds.With("service_id", supplierService.Id).Observe(duration)
	}()

	relayRequestSizeBytes.With("service_id", supplierService.Id).
		Observe(float64(relayRequest.Size()))

	relay, err := sync.serveHTTP(ctx, serviceConfig, supplierService, request, relayRequest)
	if err != nil {
		// Reply with an error if the relay could not be served.
		sync.replyWithError(ctx, requestPayload, writer, listenAddress, supplierService.Id, err)
		sync.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	// Send the relay response to the client.
	if err := sync.sendRelayResponse(relay.Res, writer); err != nil {
		sync.replyWithError(ctx, requestPayload, writer, listenAddress, supplierService.Id, err)
		sync.logger.Warn().Err(err).Msg("failed sending relay response")
		return
	}

	sync.logger.Info().Fields(map[string]any{
		"application_address":  relay.Res.Meta.SessionHeader.ApplicationAddress,
		"service_id":           relay.Res.Meta.SessionHeader.Service.Id,
		"session_start_height": relay.Res.Meta.SessionHeader.SessionStartBlockHeight,
		"server_addr":          sync.server.Addr,
	}).Msg("relay request served successfully")

	relaysSuccessTotal.With("service_id", supplierService.Id).Add(1)

	relayResponseSizeBytes.With("service_id", supplierService.Id).
		Observe(float64(relay.Res.Size()))

	// Emit the relay to the servedRelays observable.
	sync.servedRelaysProducer <- relay
}

// serveHTTP holds the underlying logic of ServeHTTP.
func (sync *synchronousRPCServer) serveHTTP(
	ctx context.Context,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
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
		Str("destination_url", serviceConfig.BackendUrl.String()).
		Msg("building relay request payload to service")

	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    serviceConfig.BackendUrl,
		Host:   serviceConfig.BackendUrl.Host,
		Body:   requestBodyReader,
	}

	if serviceConfig.Authentication != nil {
		relayHTTPRequest.SetBasicAuth(
			serviceConfig.Authentication.Username,
			serviceConfig.Authentication.Password,
		)
	}

	// Add the headers to the request.
	for key, value := range serviceConfig.Headers {
		relayHTTPRequest.Header.Add(key, value)
	}

	// Send the relay request to the native service.
	httpResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, err
	}

	responseBody, err := decodeHTTPResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}
	defer responseBody.Close()

	// Build the relay response from the native service response
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it was verified to be valid
	// and has to be the same as the relayResponse session header.
	sync.logger.Debug().
		Str("relay_request_session_header", relayRequest.Meta.SessionHeader.String()).
		Msg("building relay response protobuf from service response")

	relayResponse, err := sync.newRelayResponse(responseBody, relayRequest.Meta.SessionHeader)
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

// decodeHTTPResponseBody takes an *http.Response and returns an io.ReadCloser
// that provides access to the decoded response body. If the Content-Encoding
// indicates a supported encoding, it applies the necessary decoding.
// If the encoding is unsupported or an error occurs during decoding setup,
// it returns an error.
func decodeHTTPResponseBody(httpResponse *http.Response) (io.ReadCloser, error) {
	switch httpResponse.Header.Get("Content-Encoding") {
	case "gzip":
		return gzip.NewReader(httpResponse.Body)
	// TODO: Add other algorithms, or an alternative would be to switch to http
	// client that manages all low-level HTTP decisions for us, something like
	// https://github.com/imroc/req, https://github.com/valyala/fasthttp or
	// https://github.com/go-resty/resty
	// case "deflate":
	//     return flate.NewReader(httpResponse.Body), nil
	default:
		// No encoding or unsupported encoding, return the original body.
		return httpResponse.Body, nil
	}
}
