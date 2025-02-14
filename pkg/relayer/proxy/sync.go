package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	sdktypes "github.com/pokt-network/shannon-sdk/types"

	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

// serveHTTP holds the underlying logic of ServeHTTP.
func (server *httpServer) serveSyncRequest(
	ctx context.Context,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
	supplierServiceId string,
	relayRequest *types.RelayRequest,
	writer http.ResponseWriter,
) error {
	// Increment the relays counter.
	relaysTotal.With(
		"service_id", supplierServiceId,
		"supplier_operator_address", relayRequest.Meta.SupplierOperatorAddress,
	).Add(1)
	defer func() {
		startTime := time.Now()
		duration := time.Since(startTime).Seconds()

		// Capture the relay request duration metric.
		relaysDurationSeconds.With("service_id", supplierServiceId).Observe(duration)
	}()

	relayRequestSizeBytes.With("service_id", supplierServiceId).
		Observe(float64(relayRequest.Size()))

	// Verify the relay request signature and session.
	// TODO_TECHDEBT(red-0ne): Currently, the relayer proxy is responsible for verifying
	// the relay request signature. This responsibility should be shifted to the relayer itself.
	// Consider using a middleware pattern to handle non-proxy specific logic, such as
	// request signature verification, session verification, and response signature.
	// This would help in separating concerns and improving code maintainability.
	// See https://github.com/pokt-network/poktroll/issues/160
	if err := server.relayAuthenticator.VerifyRelayRequest(ctx, relayRequest, supplierServiceId); err != nil {
		return err
	}

	// Optimistically accumulate the relay reward before actually serving the relay.
	// The relay price will be deducted from the application's stake before the relay is served.
	// If the relay comes out to be not reward / volume applicable, the miner will refund the
	// claimed price back to the application.
	if err := server.relayMeter.AccumulateRelayReward(ctx, relayRequest.Meta); err != nil {
		return err
	}

	// Deserialize the relay request payload to get the upstream HTTP request.
	poktHTTPRequest, err := sdktypes.DeserializeHTTPRequest(relayRequest.Payload)
	if err != nil {
		return err
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

	server.logger.Debug().
		Str("destination_url", serviceConfig.BackendUrl.String()).
		Msg("building relay request payload to service")

	// Replace the upstream request URL with the host and scheme of the service
	// backend's while preserving the other components.
	// This is to ensure that the request complies with the requested service's API,
	// while being served from another location.
	requestUrl, err := url.Parse(poktHTTPRequest.Url)
	if err != nil {
		return err
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
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}
	defer httpResponse.Body.Close()

	// Serialize the service response to be sent back to the client.
	// This will include the status code, headers, and body.
	_, responseBz, err := sdktypes.SerializeHTTPResponse(httpResponse)
	if err != nil {
		return err
	}

	server.logger.Debug().
		Str("relay_request_session_header", relayRequest.Meta.SessionHeader.String()).
		Msg("building relay response protobuf from service response")

	// Build the relay response using the original service's response.
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it
	// was verified to be valid and has to be the same as the relayResponse session header.
	relayResponse, err := server.newRelayResponse(responseBz, relayRequest.Meta.SessionHeader, relayRequest.Meta.SupplierOperatorAddress)
	if err != nil {
		// The client should not have knowledge about the RelayMiner's issues with
		// building the relay response. Reply with an internal error so that the
		// original error is not exposed to the client.
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	relay := &types.Relay{Req: relayRequest, Res: relayResponse}

	// Send the relay response to the client.
	if err := server.sendRelayResponse(relay.Res, writer); err != nil {
		// If the originHost cannot be parsed, reply with an internal error so that
		// the original error is not exposed to the client.
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		server.logger.Warn().Err(err).Msg("failed sending relay response")
		return clientError
	}

	server.logger.Debug().Fields(map[string]any{
		"application_address":  relay.Res.Meta.SessionHeader.ApplicationAddress,
		"service_id":           relay.Res.Meta.SessionHeader.ServiceId,
		"session_start_height": relay.Res.Meta.SessionHeader.SessionStartBlockHeight,
		"server_addr":          server.server.Addr,
	}).Msg("relay request served successfully")

	relaysSuccessTotal.With("service_id", supplierServiceId).Add(1)

	relayResponseSizeBytes.With("service_id", supplierServiceId).
		Observe(float64(relay.Res.Size()))

	// Emit the relay to the servedRelays observable.
	server.servedRelaysProducer <- relay

	return nil
}

// sendRelayResponse marshals the relay response and sends it to the client.
func (server *httpServer) sendRelayResponse(
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
