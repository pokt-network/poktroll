package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/pokt-network/shannon-sdk/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
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

	// relayMeter is the relay meter that the RelayServer uses to meter the relays and claim the relay price.
	// It is used to ensure that the relays are metered and priced correctly.
	relayMeter relayer.RelayMeter
}

// NewSynchronousServer creates a new HTTP server that listens for incoming
// relay requests and forwards them to the supported proxied service endpoint.
// It takes the serviceId, endpointUrl, and the main RelayerProxy as arguments
// and returns a RelayServer that listens to incoming RelayRequests.
// TODO_RESEARCH(#590): Currently, the communication between the Gateway and the
// RelayMiner uses HTTP. This could be changed to a more generic and performant
// one, such as pure TCP.
func NewSynchronousServer(
	logger polylog.Logger,
	serverConfig *config.RelayMinerServerConfig,
	servedRelaysProducer chan<- *types.Relay,
	proxy relayer.RelayerProxy,
	relayMeter relayer.RelayMeter,
) relayer.RelayServer {
	return &synchronousRPCServer{
		logger:               logger,
		server:               &http.Server{Addr: serverConfig.ListenAddress},
		relayerProxy:         proxy,
		servedRelaysProducer: servedRelaysProducer,
		serverConfig:         serverConfig,
		relayMeter:           relayMeter,
	}
}

// Start starts the service server and returns an error if it fails.
// It also waits for the passed in context to end before shutting down.
// This method is blocking and should be called in a goroutine.
func (sync *synchronousRPCServer) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = sync.server.Shutdown(ctx)
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

	sync.logger.Debug().Msg("serving synchronous relay request")

	// Extract the relay request from the request body.
	sync.logger.Debug().Msg("extracting relay request from request body")
	relayRequest, err := sync.newRelayRequest(request)
	request.Body.Close()
	if err != nil {
		sync.replyWithError(err, nil, writer)
		sync.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	if err = relayRequest.ValidateBasic(); err != nil {
		sync.replyWithError(err, relayRequest, writer)
		sync.logger.Warn().Err(err).Msg("failed validating relay response")
		return
	}

	supplierServiceId := relayRequest.Meta.SessionHeader.ServiceId

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
		// If the originHost cannot be parsed, reply with an internal error so that
		// the original error is not exposed to the client.
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		sync.replyWithError(clientError, relayRequest, writer)
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
		sync.replyWithError(ErrRelayerProxyServiceEndpointNotHandled, relayRequest, writer)
		return
	}

	// Increment the relays counter.
	relaysTotal.With(
		"service_id", supplierServiceId,
		"supplier_operator_address", relayRequest.Meta.SupplierOperatorAddress,
	).Add(1)
	defer func() {
		duration := time.Since(startTime).Seconds()

		// Capture the relay request duration metric.
		relaysDurationSeconds.With("service_id", supplierServiceId).Observe(duration)
	}()

	relayRequestSizeBytes.With("service_id", supplierServiceId).
		Observe(float64(relayRequest.Size()))

	relay, err := sync.serveHTTP(ctx, serviceConfig, supplierServiceId, relayRequest)
	if err != nil {
		// Reply with an error if the relay could not be served.
		sync.replyWithError(err, relayRequest, writer)
		sync.logger.Warn().Err(err).Msg("failed serving relay request")
		return
	}

	// Send the relay response to the client.
	if err := sync.sendRelayResponse(relay.Res, writer); err != nil {
		// If the originHost cannot be parsed, reply with an internal error so that
		// the original error is not exposed to the client.
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		sync.replyWithError(clientError, relayRequest, writer)
		sync.logger.Warn().Err(err).Msg("failed sending relay response")
		return
	}

	sync.logger.Debug().Fields(map[string]any{
		"application_address":  relay.Res.Meta.SessionHeader.ApplicationAddress,
		"service_id":           relay.Res.Meta.SessionHeader.ServiceId,
		"session_start_height": relay.Res.Meta.SessionHeader.SessionStartBlockHeight,
		"server_addr":          sync.server.Addr,
	}).Msg("relay request served successfully")

	relaysSuccessTotal.With("service_id", supplierServiceId).Add(1)

	relayResponseSizeBytes.With("service_id", supplierServiceId).
		Observe(float64(relay.Res.Size()))

	// Emit the relay to the servedRelays observable.
	sync.servedRelaysProducer <- relay
}

// serveHTTP holds the underlying logic of ServeHTTP.
func (sync *synchronousRPCServer) serveHTTP(
	ctx context.Context,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
	supplierServiceId string,
	relayRequest *types.RelayRequest,
) (*types.Relay, error) {
	// Verify the relay request signature and session.
	// TODO_TECHDEBT(red-0ne): Currently, the relayer proxy is responsible for verifying
	// the relay request signature. This responsibility should be shifted to the relayer itself.
	// Consider using a middleware pattern to handle non-proxy specific logic, such as
	// request signature verification, session verification, and response signature.
	// This would help in separating concerns and improving code maintainability.
	// See https://github.com/pokt-network/poktroll/issues/160
	if err := sync.relayerProxy.VerifyRelayRequest(ctx, relayRequest, supplierServiceId); err != nil {
		return nil, err
	}

	// Optimistically accumulate the relay reward before actually serving the relay.
	// The relay price will be deducted from the application's stake before the relay is served.
	// If the relay comes out to be not reward / volume applicable, the miner will refund the
	// claimed price back to the application.
	if err := sync.relayMeter.AccumulateRelayReward(ctx, relayRequest.Meta); err != nil {
		return nil, err
	}

	// Deserialize the relay request payload to get the upstream HTTP request.
	poktHTTPRequest, err := sdktypes.DeserializeHTTPRequest(relayRequest.Payload)
	if err != nil {
		return nil, err
	}

	// Build the request to be sent to the native service by substituting
	// the destination URL's host with the native service's listen address.
	// This business logic is specific to the RelayMiner, and Gateways do not need
	// to have have knowledge of it.
	// It is the translation of the full Gateway->RelayMiner request to a
	// RelayMiner->BackendService and needs to be as transparent as possible.
	// The reply being sent back to the Gateway needs to be the same as the original,
	// "as if" the request was sent directly to the BackendService. Which means
	// the inclusion of any response headers, status codes and bodies.

	sync.logger.Debug().
		Str("destination_url", serviceConfig.BackendUrl.String()).
		Msg("building relay request payload to service")

	// Replace the upstream request URL with the host and scheme of the service
	// backend's while preserving the other components.
	// This is to ensure that the request complies with the requested service's API,
	// while being served from another location.
	requestUrl, err := url.Parse(poktHTTPRequest.Url)
	if err != nil {
		return nil, err
	}

	requestUrl.Host = serviceConfig.BackendUrl.Host
	requestUrl.Scheme = serviceConfig.BackendUrl.Scheme

	// Prepend the path of the service's backend URL to the path of the upstream request.
	// This is done to ensure that the request complies with the service's backend URL,
	// while preserving the path of the original request.
	// This is particularly important for RESTful APIs where the path is used to
	// determine the resource being accessed.
	// For example, if the service's backend URL is "http://host:8080/api/v1",
	// and the upstream request path is "/users", the final request path will be
	// "http://host:8080/api/v1/users".
	requestUrl.Path = path.Join(serviceConfig.BackendUrl.Path, requestUrl.Path)

	// Merge the query parameters of the upstream request with the query parameters
	// of the service's backend URL.
	// This is done to ensure that the query parameters of the original request are
	// passed and that the service's backend URL query parameters are also included.
	// This is important for RESTful APIs where query parameters are used to filter
	// and paginate resources.
	// For example, if the service's backend URL is "http://host:8080/api/v1?key=abc",
	// and the upstream request has a query parameter "page=1", the final request URL
	// will be "http://host:8080/api/v1?key=abc&page=1".
	query := requestUrl.Query()
	for key, values := range serviceConfig.BackendUrl.Query() {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	requestUrl.RawQuery = query.Encode()

	// TODO_TEST(red0ne): Test the request URL construction with different upstream
	// request paths and query parameters.
	// Use the same method, headers, and body as the original request to query the
	// backend URL.
	requestUrl.Host = serviceConfig.BackendUrl.Host

	// Create the HTTP headers for the request by converting the RelayRequest's
	// POKTHTTPRequest.Header to an http.Header.
	headers := http.Header{}
	poktHTTPRequest.CopyToHTTPHeader(headers)

	// Create the HTTP request out of the RelayRequest's payload.
	httpRequest := &http.Request{
		Method: poktHTTPRequest.Method,
		URL:    requestUrl,
		Header: headers,
		Body:   io.NopCloser(bytes.NewReader(poktHTTPRequest.BodyBz)),
	}
	requestUrl.RawQuery = query.Encode()

	// TODO_TEST(red0ne): Test the request URL construction with different upstream
	// request paths and query parameters.
	// Use the same method, headers, and body as the original request to query the
	// backend URL.
	httpRequest.Host = serviceConfig.BackendUrl.Host

	if serviceConfig.Authentication != nil {
		httpRequest.SetBasicAuth(
			serviceConfig.Authentication.Username,
			serviceConfig.Authentication.Password,
		)
	}

	// Add any service configuration specific headers to the request, such as
	// authentication or authorization headers. These will override any upstream
	// request headers with the same key.
	for key, value := range serviceConfig.Headers {
		httpRequest.Header.Set(key, value)
	}

	// Configure the HTTP client to use the appropriate transport based on the
	// backend URL scheme.
	var client *http.Client
	switch serviceConfig.BackendUrl.Scheme {
	case "https":
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{},
		}
		client = &http.Client{Transport: transport}
	default:
		client = http.DefaultClient
	}

	// Send the relay request to the native service.
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		// Do not expose connection errors with the backend service to the client.
		return nil, ErrRelayerProxyInternalError.Wrap(err.Error())
	}
	defer httpResponse.Body.Close()

	// Serialize the service response to be sent back to the client.
	// This will include the status code, headers, and body.
	_, responseBz, err := sdktypes.SerializeHTTPResponse(httpResponse)
	if err != nil {
		return nil, err
	}

	sync.logger.Debug().
		Str("relay_request_session_header", relayRequest.Meta.SessionHeader.String()).
		Msg("building relay response protobuf from service response")

	// Build the relay response using the original service's response.
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it
	// was verified to be valid and has to be the same as the relayResponse session header.
	relayResponse, err := sync.newRelayResponse(responseBz, relayRequest.Meta.SessionHeader, relayRequest.Meta.SupplierOperatorAddress)
	if err != nil {
		// The client should not have knowledge about the RelayMiner's issues with
		// building the relay response. Reply with an internal error so that the
		// original error is not exposed to the client.
		return nil, ErrRelayerProxyInternalError.Wrap(err.Error())
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
