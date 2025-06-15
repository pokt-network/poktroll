package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	sdktypes "github.com/pokt-network/shannon-sdk/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
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
	startTime := time.Now()
	startHeight := server.blockClient.LastBlock(ctx).Height()

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msg("handling HTTP request")

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

	// Check if the request's selected supplier is available for relaying.
	availableSuppliers := server.relayAuthenticator.GetSupplierOperatorAddresses()
	if !slices.Contains(availableSuppliers, meta.SupplierOperatorAddress) {
		logger.Warn().Msgf(
			"supplier %q operator address is not available in [%s]",
			meta.SupplierOperatorAddress,
			strings.Join(availableSuppliers, ", "),
		)
		return relayRequest, ErrRelayerProxySupplierNotReachable
	}

	// Set per-request timeouts based on the service ID configuration.
	// This overrides the server's default timeout values for this specific request.
	requestTimeout := server.requestTimeoutForServiceId(serviceId)
	rc := http.NewResponseController(writer)
	// Set write deadline: ensures the response is sent back promptly to the client.
	// If the server cannot complete sending the response within this timeout, the connection is closed.
	if err = rc.SetWriteDeadline(time.Now().Add(requestTimeout)); err != nil {
		logger.Warn().Err(err).Msg("failed setting write deadline for response controller")
		return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	// Track whether the relay completes successfully to handle reward management
	// A successful relay means that:
	// - The relay request was processed without errors
	// - The relay response was sent back to the client
	// - The relay was forwarded to the miner for mining eligibility checking
	shouldRewardRelay := false

	// Define a cleanup function to handle reward management for failed relays
	unclaimOptimisticallyAccumulatedFailedRelayReward := func() {
		if !shouldRewardRelay {
			// If the relay was not successful, revert any optimistically accumulated rewards.
			// This handles several failure scenarios such as:
			// - Request validation failures
			// - Backend connection errors
			// - Backend 5xx errors
			server.relayMeter.SetNonApplicableRelayReward(ctx, relayRequest.Meta)
		}
	}

	// Register the cleanup function to run when this function exits.
	// This ensures reward management happens regardless of how the function returns
	// (regular return or error).
	defer unclaimOptimisticallyAccumulatedFailedRelayReward()

	// Use an optimistic relay reward accumulation (before serving) for two critical reasons:
	//
	// 1. Rate Limiting:
	//    - Prevents concurrent requests from bypassing rate limits
	//    - Ensures proper accounting when multiple requests arrive simultaneously
	//
	// 2. Stake Verification:
	//    - Immediately rejects relays if the application has insufficient stake
	//    - Avoids wasting resources on requests that can't be properly rewarded
	//
	// Reward accumulation is reverted automatically when:
	//    - The relay isn't successfully completed
	//
	// This approach prioritizes accurate accounting over optimistic processing.
	//
	// TODO_CONSIDERATION: Consider implementing a delay queue instead of rejecting
	// requests when application stake is insufficient. This would allow processing
	// once earlier requests complete and free up stake.
	isOverServicing := server.relayMeter.IsOverServicing(ctx, meta)
	shouldRateLimit := isOverServicing && !server.relayMeter.AllowOverServicing()
	if shouldRateLimit {
		return relayRequest, ErrRelayerProxyRateLimited
	}

	var serviceConfig *config.RelayMinerSupplierServiceConfig

	// Get the Service and serviceUrl corresponding to the originHost.
	// TODO_IMPROVE(red-0ne): Checking that the originHost is currently done by
	// iterating over the server config's suppliers and checking if the originHost
	// is present in any of the supplier's service's hosts. We could improve this
	// by building a map at the server initialization level with originHost as the
	// key so that we can get the service and serviceUrl in O(1) time.
	for _, supplierServiceConfig := range server.serverConfig.SupplierConfigsMap {
		if serviceId == supplierServiceConfig.ServiceId {
			serviceConfig = supplierServiceConfig.ServiceConfig
			break
		}
	}

	if serviceConfig == nil {
		return relayRequest, ErrRelayerProxyServiceEndpointNotHandled.Wrapf(
			"service %q not configured",
			serviceId,
		)
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

	// If the backend service returns a 5xx error, we consider it an internal error
	// and do not expose the error to the client.
	if httpResponse.StatusCode >= 500 {
		logger.Error().
			Int("status_code", httpResponse.StatusCode).
			Msg("backend service returned a server error")

		return relayRequest, ErrRelayerProxyInternalError.Wrapf(
			"backend service returned an error with status code %d",
			httpResponse.StatusCode,
		)
	}

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

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msg("relay request served successfully")

	relayer.RelaysSuccessTotal.With("service_id", serviceId).Add(1)

	relayer.RelayResponseSizeBytes.With("service_id", serviceId).Observe(float64(relay.Res.Size()))

	// Verify relay reward eligibility again after completing the backend request.
	// During long-running backend requests (especially under high load or with LLM services),
	// the session may have expired while we were waiting for the response.
	// If a session expires during processing, the relay is classified as "over-servicing"
	// and becomes ineligible for rewards, as it falls outside the protocol's reward mechanism.
	if err := server.relayAuthenticator.CheckRelayRewardEligibility(ctx, relayRequest); err != nil {
		processingTime := time.Since(startTime).Milliseconds()
		logger.Warn().Msgf(
			"relay request is no longer eligible for rewards, request took %d ms, starting at block %d and ending at block %d: %v",
			processingTime, server.blockClient.LastBlock(ctx).Height(), startHeight, err,
		)

		isOverServicing = true
	}

	// Only emit relays and mark them as rewardable when they are not over-servicing:
	// - Over-serviced relays exceed the application's allocated stake.
	// - These are provided as free goodwill by the supplier.
	// - Not eligible for on-chain compensation (outside protocol's reward mechanism).
	//
	// Emitting over-serviced relays would:
	// - Break the optimistic relay reward accumulation pattern.
	// - Mix "goodwill service" with "protocol-compensated service".
	//
	// Protocol details:
	// - Relay rewards are optimistically accumulated before forwarding to the relay miner.
	// - Over-serviced relays must never enter this reward pipeline.
	if !isOverServicing {
		// Forward reward-eligible relays for SMT updates (excludes over-serviced relays).
		server.servedRewardableRelaysProducer <- relay

		// Mark relay as successful and rewardable, so deferred logic doesn't revert it.
		shouldRewardRelay = true
	}

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
