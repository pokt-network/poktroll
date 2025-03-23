package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	validate "github.com/go-playground/validator/v10"
	sdktypes "github.com/pokt-network/shannon-sdk/types"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

// serveSyncRequest serves a synchronous relay request by forwarding the request
// to the service's backend URL and returning the response to the client.
func (server *relayMinerHTTPServer) serveSyncRequest(
	ctx context.Context,
	writer http.ResponseWriter,
	request *http.Request,
) (*types.RelayRequest, error) {
	logger := server.logger.With("relay_request_type", "synchronous")

	logger.Debug().Msg("handling HTTP request")

	// Extract the relay request from the request body.
	logger.Debug().Msg("extracting relay request from request body")
	relayRequest, err := server.newRelayRequest(request)
	if err != nil {
		logger.Warn().Err(err).Msg("failed creating relay request")
		return relayRequest, err
	}
	request.Body.Close()

	if err = relayRequest.ValidateBasic(); err != nil {
		logger.Warn().Err(err).Msg("failed validating relay request")
		return relayRequest, err
	}

	meta := relayRequest.Meta
	serviceId := meta.SessionHeader.ServiceId

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
		return relayRequest, clientError
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
			if host == originHostUrl.Hostname() && serviceId == supplierServiceConfig.ServiceId {
				serviceConfig = supplierServiceConfig.ServiceConfig
				break
			}
		}

		if serviceConfig != nil {
			break
		}
	}

	if serviceConfig == nil {
		return relayRequest, ErrRelayerProxyServiceEndpointNotHandled
	}

	logger = logger.With(
		"service_id", serviceId,
		"server_addr", server.server.Addr,
		"application_address", meta.SessionHeader.ApplicationAddress,
		"session_start_height", meta.SessionHeader.SessionStartBlockHeight,
		"destination_url", serviceConfig.BackendUrl.String(),
	)

	// Increment the relays counter.
	relayer.RelaysTotal.With(
		"service_id", serviceId,
		"supplier_operator_address", meta.SupplierOperatorAddress,
	).Add(1)
	defer func() {
		startTime := time.Now()
		duration := time.Since(startTime).Seconds()

		// Capture the relay request duration metric.
		relayer.RelaysDurationSeconds.With("service_id", serviceId).Observe(duration)
	}()

	relayer.RelayRequestSizeBytes.With("service_id", serviceId).
		Observe(float64(relayRequest.Size()))

	// Verify the relay request signature and session.
	if err = server.relayAuthenticator.VerifyRelayRequest(ctx, relayRequest, serviceId); err != nil {
		return relayRequest, err
	}

	// Optimistically accumulate the relay reward before actually serving the relay.
	// The relay price will be deducted from the application's stake before the relay is served.
	// If the relay comes out to be not reward / volume applicable, the miner will refund the
	// claimed price back to the application.
	if err = server.relayMeter.AccumulateRelayReward(ctx, meta); err != nil {
		return relayRequest, err
	}

	httpRequest, err := relayer.BuildServiceBackendRequest(relayRequest, serviceConfig)
	if err != nil {
		logger.Error().Err(err).Msg("failed to build the service backend request")
		return relayRequest, ErrRelayerProxyInternalError.Wrapf("failed to build the service backend request: %v", err)
	}
	defer httpRequest.Body.Close()

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
		return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	}
	defer httpResponse.Body.Close()

	// Serialize the service response to be sent back to the client.
	// This will include the status code, headers, and body.
	_, responseBz, err := sdktypes.SerializeHTTPResponse(httpResponse)
	if err != nil {
		return relayRequest, err
	}

	logger.Debug().
		Str("relay_request_session_header", meta.SessionHeader.String()).
		Msg("building relay response protobuf from service response")

	// Build the relay response using the original service's response.
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it
	// was verified to be valid and has to be the same as the relayResponse session header.
	relayResponse, err := server.newRelayResponse(responseBz, meta.SessionHeader, meta.SupplierOperatorAddress)
	if err != nil {
		// The client should not have knowledge about the RelayMiner's issues with
		// building the relay response. Reply with an internal error so that the
		// original error is not exposed to the client.
		return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	relay := &types.Relay{Req: relayRequest, Res: relayResponse}

	// Send the relay response to the client.
	if err = server.sendRelayResponse(relay.Res, writer); err != nil {
		// If the originHost cannot be parsed, reply with an internal error so that
		// the original error is not exposed to the client.
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		logger.Warn().Err(err).Msg("failed sending relay response")
		return relayRequest, clientError
	}

	logger.Debug().Msg("relay request served successfully")

	relayer.RelaysSuccessTotal.With("service_id", serviceId).Add(1)

	relayer.RelayResponseSizeBytes.With("service_id", serviceId).
		Observe(float64(relay.Res.Size()))

	// Emit the relay to the servedRelays observable.
	server.servedRelaysProducer <- relay

	return relayRequest, nil
}

// sendRelayResponse marshals the relay response and sends it to the client.
func (server *relayMinerHTTPServer) sendRelayResponse(
	relayResponse *types.RelayResponse,
	writer http.ResponseWriter,
) error {
	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		return err
	}

	relayResponseBzLenStr := fmt.Sprintf("%d", len(relayResponseBz))

	// Send the close and content length headers to the client to ensure that the
	// connection is closed after the response is sent.
	// This should be done automatically by the http server but they are set to
	// ensure deterministic behavior.
	writer.Header().Set("Connection", "close")
	writer.Header().Set("Content-Length", relayResponseBzLenStr)
	_, err = writer.Write(relayResponseBz)
	return err
}

// forwardPayload represents the HTTP request body format to forward a request
// to the supplier.
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

// Validate returns true if the payload format is correct based on the
// value validation rules.
func (p forwardPayload) Validate() error {
	var err error
	if structErr := validate.New().Struct(&p); structErr != nil {
		for _, e := range structErr.(validate.ValidationErrors) {
			err = errors.Join(err, e)
		}
	}

	return err
}

// forwardHTTP forward a HTTP request:
// - It reads the entire payload from the client.
// - It validates the input payload.
// - It sends the request to the supplier backend URL.
// - It streams back the response to the client.
func (server *relayMinerHTTPServer) forwardHTTP(ctx context.Context, supplierConfig *config.RelayMinerSupplierConfig, w http.ResponseWriter, req *http.Request) error {
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
			"service_id": supplierConfig.ServiceId,
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
			"service_id": supplierConfig.ServiceId,
			"method":     payload.Method,
			"path":       payload.Path,
			"headers":    payload.Headers,
		}).Err(err).Msg("failed to write forward http reponse")

		http.Error(w, fmt.Sprintf("relayminer: error on forward http response: %s", err.Error()), http.StatusInternalServerError)

		return err
	}

	return nil
}
