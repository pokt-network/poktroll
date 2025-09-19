package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	// writeDeadlineSafetyDuration provides extra buffer time beyond the request timeout
	// to ensure the HTTP response can be fully written before the connection is closed.
	// This prevents incomplete responses due to network write timing issues.
	writeDeadlineSafetyDuration = 1 * time.Second
	// Fallback timeout for request preparation exceeding service timeout limits.
	fallbackTimeout = 1 * time.Second
)

// serveSyncRequest serves a synchronous relay request by forwarding the request
// to the service's backend URL and returning the response to the client.
func (server *relayMinerHTTPServer) serveSyncRequest(
	ctx context.Context,
	writer http.ResponseWriter,
	request *http.Request,
) (*types.RelayRequest, error) {
	// Default to a failure (5XX).
	// Success is implied by reaching the end of the function where status is set to 2XX.
	statusCode := http.StatusInternalServerError
	// Ensure the context is set with the proxy component kind.
	// This is used to capture the component kind in gRPC call duration metrics collection.
	ctx = context.WithValue(ctx, query.ComponentCtxRelayMinerKey, query.ComponentCtxRelayMinerProxy)

	logger := server.logger.With(
		"relay_request_type", "⚡ synchronous",
		"rpc_type", request.Header.Get(RPCTypeHeader),
	)
	requestStartTime := time.Now()
	startBlock := server.blockClient.LastBlock(ctx)
	startHeight := startBlock.Height()

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) at relay request start",
		startHeight,
		startBlock.Hash(),
	)

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msg("handling HTTP request")

	// Extract the relay request from the request body.
	logger.Debug().Msg("extracting relay request from request body")
	relayRequest, relayRequestRelease, err := server.newRelayRequest(request)
	if err != nil {
		logger.Warn().Err(err).Msg("❌ Failed creating relay request")
		return relayRequest, err
	}
	defer relayRequestRelease()

	if err = relayRequest.ValidateBasic(); err != nil {
		logger.Warn().Err(err).Msg("❌ Failed validating relay request")
		return relayRequest, err
	}

	meta := relayRequest.Meta
	sessionHeader := meta.SessionHeader
	serviceId := sessionHeader.ServiceId

	logger = logger.With(
		"current_height", startHeight,
		"session_id", sessionHeader.SessionId,
		"session_start_height", sessionHeader.SessionStartBlockHeight,
		"session_end_height", sessionHeader.SessionEndBlockHeight,
		"service_id", serviceId,
		"application_address", sessionHeader.ApplicationAddress,
		"supplier_operator_address", meta.SupplierOperatorAddress,
		"request_start_time", requestStartTime.String(),
	)

	// Check if the request's selected supplier is available for relaying.
	availableSuppliers := server.relayAuthenticator.GetSupplierOperatorAddresses()

	if !slices.Contains(availableSuppliers, meta.SupplierOperatorAddress) {
		logger.Warn().
			Msgf(
				"❌ The request's selected supplier with operator_address (%q) is not available for relaying! "+
					"This could be a network or configuration issue. Available suppliers: [%s] 🚦",
				meta.SupplierOperatorAddress,
				strings.Join(availableSuppliers, ", "),
			)
		return relayRequest, ErrRelayerProxySupplierNotReachable
	}

	// Set per-request timeouts based on the service ID configuration.
	// This overrides the server's default timeout values for this specific request.
	requestTimeout := server.requestTimeoutForServiceId(serviceId)

	// Calculate the absolute requestDeadline for this request processing cycle.
	// Includes both the service request timeout and additional buffer for response writing.
	requestDeadline := time.Now().Add(requestTimeout + writeDeadlineSafetyDuration)
	logger = logger.With("deadline", requestDeadline)

	ctxWithDeadline, cancel := context.WithDeadline(ctx, requestDeadline)
	defer cancel()

	// This is important to ensure that the server's timeout defaults are overridden
	// by the request-specific timeout.
	rc := http.NewResponseController(writer)
	// Set a write deadline for the HTTP response writer to prevent hanging connections.
	// The deadline includes an additional safety buffer to ensure the response can be written.
	if err = rc.SetWriteDeadline(requestDeadline.Add(writeDeadlineSafetyDuration)); err != nil {
		logger.Warn().Err(err).Msg("failed setting write deadline for response controller")
		return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	// TODO_TECHDEBT: Consider re-enabling ResponseController write deadlines
	// after investigating potential compatibility issues with the current setup.
	// The commented code below was intended to ensure timely response delivery:
	//
	// rc := http.NewResponseController(writer)
	// if err = rc.SetWriteDeadline(deadline.Add(writeDeadlineSafetyDelta)); err != nil {
	// 	logger.Warn().Err(err).Msg("failed setting write deadline for response controller")
	// 	return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	// }

	// Track whether the relay completes successfully to handle reward management.
	// A successful relay means that:
	// - The relay request was processed without errors
	// - The relay response was sent back to the client
	// - The relay was forwarded to the miner for mining eligibility checking
	shouldRewardRelay := false

	// Track whether relay rewards have been optimistically accumulated for this request.
	// Used to determine if rewards need to be reverted on failure.
	isRelayRewardAccumulated := false

	// Define a cleanup function to handle reward management for failed relays.
	unclaimOptimisticallyAccumulatedFailedRelayReward := func() {
		if !shouldRewardRelay && isRelayRewardAccumulated {
			// Revert any optimistically accumulated rewards when relay fails.
			// This covers failure scenarios:
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

	// Use optimistic relay reward accumulation (before serving) for:
	//
	// 1. Rate Limiting:
	//    - Prevents concurrent requests from bypassing rate limits
	//    - Ensures proper accounting when multiple requests arrive simultaneously
	//
	// 2. Stake Verification:
	//    - Immediately rejects relays if the application has insufficient stake
	//    - Avoids wasting resources on requests that can't be properly rewarded
	//
	// Reward accumulation is reverted automatically when the relay isn't successfully completed.
	// This approach prioritizes accurate accounting over optimistic processing.
	//
	// TODO_CONSIDERATION: Consider implementing a delay queue instead of rejecting
	// requests when application stake is insufficient. This would allow processing
	// once earlier requests complete and free up stake.
	isOverServicing := false
	// Check whether the relay's session is already known and its corresponding data cached.
	isSessionKnown := server.isSessionKnown(sessionHeader.SessionId)
	// Perform relay request checks and validation only if the session is known
	// or if eager validation is enabled.
	if isSessionKnown || server.eagerValidationEnabled {
		isOverServicing = server.relayMeter.IsOverServicing(ctxWithDeadline, meta)
		shouldRateLimit := isOverServicing && !server.relayMeter.AllowOverServicing()
		if shouldRateLimit {
			return relayRequest, ErrRelayerProxyRateLimited
		}

		// Ensure the session is known and eager validation is active for the current request.
		server.markSessionAsKnown(sessionHeader.SessionId, sessionHeader.SessionEndBlockHeight)
		isSessionKnown = true
	}

	// Mark that relay rewards have been optimistically accumulated.
	// This flag enables the cleanup function to revert rewards if the relay fails.
	isRelayRewardAccumulated = true

	// Get the supplier config for the service.
	supplierConfig, ok := server.serverConfig.SupplierConfigsMap[serviceId]
	if !ok {
		return relayRequest, ErrRelayerProxyServiceEndpointNotHandled.Wrapf(
			"service %q not configured",
			serviceId,
		)
	}

	// Get the service config from the supplier config.
	// This will use either the RPC type specific service config or the default service config.
	serviceConfig, serviceConfigTypeLog, err := getServiceConfig(logger, supplierConfig, request)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed getting service config")
		return relayRequest, err
	}

	if serviceConfig == nil {
		return relayRequest, ErrRelayerProxyServiceEndpointNotHandled.Wrapf(
			"service %q not configured",
			serviceId,
		)
	}

	// Hydrate the logger with relevant values.
	logger = logger.With(
		"server_addr", server.server.Addr,
		"destination_url", serviceConfig.BackendUrl.String(),
		"service_config_type", serviceConfigTypeLog,
	)

	// Increment the relays counter.
	relayer.RelaysTotal.With(
		"service_id", serviceId,
		"supplier_operator_address", meta.SupplierOperatorAddress,
	).Add(1)
	defer func(startTime time.Time, statusCode *int) {
		// Capture the relay request duration metric.
		relayer.CaptureRelayDuration(serviceId, startTime, *statusCode)
	}(requestStartTime, &statusCode)

	relayer.RelayRequestSizeBytes.With("service_id", serviceId).
		Observe(float64(relayRequest.Size()))

	// Verify the relay request signature and session when:
	// 1. The session is already known (cached)
	// 2. Eager validation is enabled (immediate validation for all requests)
	if isSessionKnown || server.eagerValidationEnabled {
		if err = server.relayAuthenticator.VerifyRelayRequest(ctxWithDeadline, relayRequest, serviceId); err != nil {
			logger.Error().Err(err).Msg("❌ Failed verifying relay request")
			return relayRequest, err
		}
	}

	httpRequest, err := relayer.BuildServiceBackendRequest(relayRequest, serviceConfig)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed building the service backend request")
		return relayRequest, ErrRelayerProxyInternalError.Wrapf("failed to build the service backend request: %v", err)
	}
	defer CloseBody(logger, httpRequest.Body)

	// Configure HTTP client based on backend URL scheme.
	var client http.Client
	switch serviceConfig.BackendUrl.Scheme {
	case "https":
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{},
		}
		client = http.Client{Transport: transport}
	default:
		// Copy default client to avoid modifying global instance.
		// Prevents race conditions from concurrent timeout modifications.
		client = *http.DefaultClient
	}

	logger = logger.With("request_preparation_duration", time.Since(requestStartTime).String())
	relayer.CaptureRequestPreparationDuration(serviceId, requestStartTime)

	// Check if context deadline already exceeded before making the backend call.
	// Prevents unnecessary work when request has already timed out.
	//
	// DEV_NOTE: Even after deadline, client cancellation or request timeout,
	//  the request handler's goroutine will continue processing unless explicitly
	//  checking for context cancellation.
	if ctxErr := ctxWithDeadline.Err(); ctxErr != nil {
		logger.With("current_time", time.Now()).Warn().Msg(ctxErr.Error())

		return relayRequest, ErrRelayerProxyTimeout.Wrapf(
			"request to service %s timed out after %s",
			serviceId,
			requestTimeout.String(),
		)
	}

	// Set HTTP client timeout to match remaining request budget.
	// Subtract preparation time from total timeout to avoid exceeding limit.
	remainingTimeout := requestTimeout - time.Since(requestStartTime)
	if remainingTimeout <= 0 {
		logger.Warn().
			Dur("request_timeout", requestTimeout).
			Dur("preparation_time", time.Since(requestStartTime)).
			Msg("Request preparation exceeded timeout. Providing additional time.")
		remainingTimeout = fallbackTimeout
	}
	client.Timeout = remainingTimeout

	// Send the relay request to the native service.
	serviceCallStartTime := time.Now()
	var httpResponse *http.Response
	// Treat "hey" as a special service: return a static JSON response without
	// calling the backend, since we're not testing backend service latency here.
	if serviceId == "hey" {
		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("{\"jsonrpc\": \"2.0\", \"result\": \"0x6942\"}")),
		}
		err = nil
	} else {
		httpResponse, err = client.Do(httpRequest)
	}

	backendServiceProcessingEnd := time.Now()
	// Add response preparation duration to the logger such that any log before errors will have
	// as much request duration information as possible.
	logger = logger.With(
		"backend_request_duration", time.Since(serviceCallStartTime).String(),
	)

	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed sending the relay request to the native service")
		// Capture the service call request duration metric.
		relayer.CaptureServiceDuration(serviceId, serviceCallStartTime, statusCode)

		// Check if error is a backend timeout.
		// URL errors with timeout flag indicate backend exceeded response time limit.
		if isTimeoutError(err) {
			logger.With("current_time", time.Now()).Warn().Msg(err.Error())
			return relayRequest, ErrRelayerProxyTimeout.Wrapf(
				"request to service %s timed out after %s",
				serviceId,
				requestTimeout.String(),
			)
		}

		// Do not expose connection errors with the backend service to the client.
		return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	defer CloseBody(logger, httpResponse.Body)
	// Capture the service call request duration metric.
	relayer.CaptureServiceDuration(serviceId, serviceCallStartTime, httpResponse.StatusCode)

	// Serialize the service response to be sent back to the client.
	// This will include the status code, headers, and body.
	// wrappedHTTPResponse, responseBz, err := SerializeHTTPResponse(logger, httpResponse, server.serverConfig.MaxBodySize)
	wrappedHTTPResponse, responseBz, payloadHash, err := SerializeHTTPResponseWithHash(logger, httpResponse, server.serverConfig.MaxBodySize)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed serializing the service response")
		return relayRequest, err
	}

	// Pass through all backend responses including errors.
	// Allows clients to see real HTTP status codes from backend service.
	// Log non-2XX status codes for monitoring but don't block response.
	if httpResponse.StatusCode >= http.StatusMultipleChoices {
		logger.Error().
			Int("status_code", httpResponse.StatusCode).
			Str("request_url", httpRequest.URL.String()).
			Str("request_payload_first_bytes", polylog.Preview(string(relayRequest.Payload))).
			Str("response_payload_first_bytes", polylog.Preview(string(wrappedHTTPResponse.BodyBz))).
			Msg("backend service returned a non-2XX status code. Passing it through to the client.")
	}

	logger.Debug().
		Str("relay_request_session_header", sessionHeader.String()).
		Msg("building relay response protobuf from service response")

	// Check context cancellation before building relay response to prevent signature race conditions
	if ctxErr := ctxWithDeadline.Err(); ctxErr != nil {
		logger.Warn().Err(ctxErr).Msg("⚠️ Context canceled before building relay response - preventing signature race condition")
		return relayRequest, ErrRelayerProxyTimeout.Wrapf(
			"request context canceled during response building: %v",
			ctxErr,
		)
	}

	// Build the relay response using the original service's response.
	// Use relayRequest.Meta.SessionHeader on the relayResponse session header since it
	// was verified to be valid and has to be the same as the relayResponse session header.
	// PERF_NOTE: Added this to avoid reading again the payload to hash it, now the hash is generated during the serialization.
	relayResponse, relayResponseRelease, err := server.newRelayResponseWithHash(responseBz, payloadHash, sessionHeader, meta.SupplierOperatorAddress)
	//relayResponse, err := server.newRelayResponse(responseBz, sessionHeader, meta.SupplierOperatorAddress)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed building the relay response")
		// The client should not have knowledge about the RelayMiner's issues with
		// building the relay response. Reply with an internal error so that the
		// original error is not exposed to the client.
		return relayRequest, ErrRelayerProxyInternalError.Wrap(err.Error())
	}
	defer relayResponseRelease()

	relay := &types.Relay{Req: relayRequest, Res: relayResponse}

	// Capture the time after response time for the relay.
	responsePreparationEnd := time.Now()
	// Add response preparation duration to the logger such that any log before errors will have
	// as much request duration information as possible.
	logger = logger.With(
		"response_preparation_duration",
		time.Since(backendServiceProcessingEnd).String(),
	)
	relayer.CaptureResponsePreparationDuration(serviceId, backendServiceProcessingEnd)

	// Send the relay response to the client.
	err = server.sendRelayResponse(relay.Res, writer)
	logger = logger.With("send_response_duration", time.Since(responsePreparationEnd).String())
	if err != nil {
		// If the originHost cannot be parsed, reply with an internal error so that
		// the original error is not exposed to the client.
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		// Log current time to highlight writer i/o timeout errors.
		logger.Warn().Err(err).Time("current_time", time.Now()).Msg("❌ Failed sending relay response")
		return relayRequest, clientError
	}

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msg("relay request served successfully")

	relayer.RelaysSuccessTotal.With("service_id", serviceId).Add(1)

	relayer.RelayResponseSizeBytes.With("service_id", serviceId).Observe(float64(relay.Res.Size()))

	// In case the current request is not validated yet perform a late validation
	// before mining the relay.
	// DEV_NOTE: If eager validation is enabled, then the session is already known.
	if !isSessionKnown {
		relayer.CaptureDelayedValidationOccurrence(serviceId)

		logger.Info().Msg("🔄 Performing delayed validation - session was unknown at request time")

		isOverServicing = server.relayMeter.IsOverServicing(ctxWithDeadline, meta)
		shouldRateLimit := isOverServicing && !server.relayMeter.AllowOverServicing()
		if shouldRateLimit {
			relayer.CaptureDelayedValidationRateLimiting(serviceId)

			return relayRequest, ErrRelayerProxyRateLimited
		}

		if err = server.relayAuthenticator.VerifyRelayRequest(ctxWithDeadline, relayRequest, serviceId); err != nil {
			logger.Error().Err(err).Msg("❌ Failed delayed validation - relay request verification failed after successful response")
			relayer.CaptureDelayedValidationFailure(serviceId)
			return relayRequest, err
		}

		// Mark the session as known to skip late validations for subsequent requests.
		server.markSessionAsKnown(sessionHeader.SessionId, sessionHeader.SessionEndBlockHeight)
	}

	// Verify relay reward eligibility a SECOND time AFTER completing backend request.
	//
	// Why needed:
	// - Session may end during long-running backend requests
	// - Examples: high load, short sessions, slow services (e.g. LLM)
	//
	// Result:
	// - Relay classified as "over-servicing"
	// - Becomes reward ineligible
	//
	// Mitigations:
	// - Longer sessions (onchain gov param)
	// - Allow over-servicing (relayminer config, still reward ineligible)
	// - Increase claim window open offset blocks (onchain gov param)
	//
	// TODO(@Olshansk): Revisit params to enable the above.
	if err := server.relayAuthenticator.CheckRelayRewardEligibility(ctx, relayRequest); err != nil {
		processingTime := time.Since(requestStartTime).Milliseconds()
		endBlock := server.blockClient.LastBlock(ctx)
		endHeight := endBlock.Height()
		logger.Warn().Msgf(
			"⏱️ Backend took %d ms — relay no longer eligible (session expired: block %d → %d, hash: %X). "+
				"Likely long response time, session too short, or full node sync issues. "+
				"Please verify your full node is in sync and not overwhelmed with websocket connections. Error: %v",
			processingTime, startHeight, endHeight, endBlock.Hash(), err,
		)

		isOverServicing = true
	}

	// Only emit relays and mark as rewardable when no over-servicing or server error:
	if isRewardApplicable(isOverServicing, httpResponse.StatusCode) {
		// Forward reward-eligible relays for SMT updates (excludes over-serviced relays).
		// We use a non-blocking select to prevent relay response delays.
		//
		// DEV_NOTE: This change was added under the presumption that a slow or full channel was resulting
		// in "missing supplier operator signature" errors.
		select {
		case server.servedRewardableRelaysProducer <- relay:
			// Successfully forwarded relay for mining
			shouldRewardRelay = true
		default:
			// Channel is full - log warning but don't block the response
			// This prevents signature validation timeouts that cause "missing supplier operator signature" errors
			logger.Warn().Msg("⚠️ Relay mining channel full - dropping relay from mining pipeline (prevents signature timeout)")
			// Don't mark as rewardable since it wasn't forwarded to miner
		}
	}

	// set to 200 because everything is good about the processed relay.
	statusCode = http.StatusOK
	return relayRequest, nil
}

// isRewardApplicable checks if the current relay is reward applicable given
// its over-servicing status and the HTTP status code of the response.
//
// - Over-serviced relays exceed application's allocated stake
// - Provided as free goodwill by supplier
// - Not eligible for on-chain compensation
//
// Emitting over-serviced relays would:
// - Break optimistic relay reward accumulation pattern
// - Mix "goodwill service" with "protocol-compensated service"
//
// Protocol details:
// - Relay rewards optimistically accumulated before forwarding to relay miner
// - Over-serviced relays MUST NOT enter reward pipeline
// - 5xx errors MUST NOT enter reward pipeline
func isRewardApplicable(isOverServicing bool, statusCode int) bool {
	// Reward is applicable when:
	// - Not over-servicing (application has enough stake)
	// - Status code is 2xx (successful relay)
	return !isOverServicing && statusCode < http.StatusInternalServerError
}

// serviceConfigTypeDefault indicates that the service config being used is
// the default service config, as opposed to a service-specific config.
const logServiceConfigTypeDefault = "DEFAULT_SERVICE_CONFIG"

// getServiceConfig returns the service config for the service.
// This will use either the RPC type specific service config or the default service config.
func getServiceConfig(
	logger polylog.Logger,
	supplierConfig *config.RelayMinerSupplierConfig,
	request *http.Request,
) (
	serviceConfig *config.RelayMinerSupplierServiceConfig,
	serviceConfigTypeLog string,
	err error,
) {
	// If the following are true:
	// 	- The RPC-type is set for the service
	// 	- The RPC-type service-specific config is available
	// Then, use the RPC-type service-specific config.
	// Otherwise, use the default service config.
	rpcTypeHeaderValue := request.Header.Get(RPCTypeHeader)

	if rpcTypeHeaderValue != "" {
		// Attempt to convert string header value to int32.
		// For example, "1" -> RPCType_GRPC, "2" -> RPCType_WEBSOCKET, etc.
		rpcTypeInt, err := strconv.Atoi(rpcTypeHeaderValue)
		if err != nil {
			return nil, "", ErrRelayerProxyInternalError.Wrapf(
				"❌ Unable to parse rpc type header value %q",
				rpcTypeHeaderValue,
			)
		}

		// If the header is successfully parsed, use the RPC type specific service config.
		rpcType := sharedtypes.RPCType(rpcTypeInt)
		if rpcTypeServiceConfig, ok := supplierConfig.RPCTypeServiceConfigs[rpcType]; ok {
			logger.Debug().Msgf("🟢 Using '%s' RPC type specific service config for service %q",
				rpcType.String(), supplierConfig.ServiceId,
			)

			// Add the RPC type to the log service config type.
			//   - eg. "JSON_RPC_SERVICE_CONFIG"
			logServiceConfigTypeRPCType := fmt.Sprintf("%s_SERVICE_CONFIG", rpcType.String())

			return rpcTypeServiceConfig, logServiceConfigTypeRPCType, nil
		} else {
			logger.Info().Msgf("ℹ️️ No '%s' RPC type specific service config found for service %q, falling back to default service config",
				rpcType.String(), supplierConfig.ServiceId,
			)
		}
	}

	logger.Debug().Msgf("🟢 Using default service config for service %q", supplierConfig.ServiceId)

	// If the RPC type is not set, use the default service config.
	return supplierConfig.ServiceConfig, logServiceConfigTypeDefault, nil
}

// sendRelayResponse marshals the relay response and sends it to the client.
func (server *relayMinerHTTPServer) sendRelayResponse(
	relayResponse *types.RelayResponse,
	writer http.ResponseWriter,
) error {
	// Double-check that the signature is present before marshaling for client.
	// DEV_NOTE: This is a secondary sanity check to avoid missing supplier signature errors.
	if len(relayResponse.Meta.GetSupplierOperatorSignature()) == 0 {
		return ErrRelayerProxyInternalError.Wrap("relay response missing supplier operator signature before marshaling - signature was lost during processing")
	}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		return err
	}

	relayResponseBzLenStr := fmt.Sprintf("%d", len(relayResponseBz))

	// Send close and content length headers to ensure connection closure
	// after response is sent. Set explicitly for deterministic behavior.
	writer.Header().Set("Connection", "close")
	writer.Header().Set("Content-Length", relayResponseBzLenStr)
	_, err = writer.Write(relayResponseBz)
	return err
}

// isSessionKnown checks if the session ID is already known and that all its
// relevant data is cached to skip late validations or fetching data again.
func (server *relayMinerHTTPServer) isSessionKnown(sessionId string) bool {
	server.knownSessionsMutex.RLock()
	defer server.knownSessionsMutex.RUnlock()
	_, ok := server.knownSessions[sessionId]

	return ok
}

// markSessionAsKnown marks the session ID as known to avoid late validations
// for subsequent requests within the same session.
func (server *relayMinerHTTPServer) markSessionAsKnown(sessionId string, sessionEndBlockHeight int64) {
	server.knownSessionsMutex.Lock()
	defer server.knownSessionsMutex.Unlock()
	server.knownSessions[sessionId] = sessionEndBlockHeight
}

// pruneOutdatedKnownSessions removes known sessions that have ended before
// the current block height to free up memory and keep the known sessions map
// up-to-date.
func (server *relayMinerHTTPServer) pruneOutdatedKnownSessions(ctx context.Context, block client.Block) {
	// TODO_TECHDEBT: Do not prune at each block, instead do it periodically each num blocks per session.
	server.knownSessionsMutex.Lock()
	defer server.knownSessionsMutex.Unlock()

	for sessionId, endHeight := range server.knownSessions {
		// TODO_TECHDEBT: Use grace period blocks instead of +1
		if endHeight+1 < block.Height() {
			delete(server.knownSessions, sessionId)
		}
	}
}

// isTimeoutError checks if the error is a timeout error.
func isTimeoutError(err error) bool {
	// Check if the error is a context deadline exceeded error.
	// This is used to determine if the request timed out.
	urlErr, ok := err.(*url.Error)
	if ok && urlErr.Timeout() {
		return true
	}
	return false
}
