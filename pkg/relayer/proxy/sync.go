package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

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
	w http.ResponseWriter,
	r *http.Request,
) (*types.RelayRequest, error) {
	start := time.Now()
	statusCode := http.StatusInternalServerError

	// default metric labels (never empty)
	supplierOperatorAddress := relayer.UnknownSupplierOperatorAddress
	serviceId := relayer.UnknownServiceID

	// attach perf tracker now that we know the serviceId
	ctx, tr, flush := relayer.EnsurePerfBuffered(ctx, serviceId)
	defer flush()

	defer func(startTime time.Time, statusCode *int) {
		tr.Start("deferred_prometheus_metrics")
		relayer.RelaysTotal.With(
			"service_id", serviceId,
			"supplier_operator_address", supplierOperatorAddress,
		).Add(1)
		relayer.CaptureRelayDuration(serviceId, startTime, *statusCode)
		tr.Finish("deferred_prometheus_metrics")
	}(start, &statusCode)

	logger := server.logger.With(
		"relay_request_type", "⚡ synchronous",
		"rpc_type", r.Header.Get(RPCTypeHeader),
	)
	ctx = context.WithValue(ctx, query.ComponentCtxRelayMinerKey, query.ComponentCtxRelayMinerProxy)

	// --- 1) Parse + basic validate ---
	tr.Start(relayer.InstructionProxySyncParseRequest)
	relayReq, err := server.parseRelayRequest(r, logger)
	tr.Finish(relayer.InstructionProxySyncParseRequest)
	if err != nil {
		return relayReq, err
	}

	meta := relayReq.Meta
	sessionHeader := meta.SessionHeader
	supplierOperatorAddress = meta.SupplierOperatorAddress
	serviceId = sessionHeader.ServiceId
	tr.SetServiceID(serviceId)

	// supplier reachability
	available := server.relayAuthenticator.GetSupplierOperatorAddresses()
	if !slices.Contains(available, supplierOperatorAddress) {
		logger.Warn().Msgf("❌ supplier %q unavailable; available: [%s]",
			supplierOperatorAddress, strings.Join(available, ", "))
		return relayReq, ErrRelayerProxySupplierNotReachable
	}

	// --- 2) Per-service config FIRST (so eager can be per-service) ---
	tr.Start(relayer.InstructionProxySyncGetServiceConfig)
	serviceConfig, serviceConfigType, err := server.resolveServiceConfig(logger, serviceId, r)
	tr.Finish(relayer.InstructionProxySyncGetServiceConfig)
	if err != nil {
		return relayReq, err
	}
	eagerEnabled := serviceConfig.EnableEagerRelayRequestValidation

	logger = logger.With(
		"server_addr", server.server.Addr,
		"destination_url", serviceConfig.BackendUrl.String(),
		"service_config_type", serviceConfigType,
	)

	relayer.RelayRequestSizeBytes.
		With("service_id", serviceId).
		Observe(float64(relayReq.Size()))

	// --- 3) Session fast-path via supervisor cache ---
	tr.Start(relayer.InstructionProxySyncGetSessionEntry)
	sessionCacheEntry, sessionKnown := server.miningSupervisor.GetSessionEntry(sessionHeader.SessionId)
	tr.Finish(relayer.InstructionProxySyncGetSessionEntry)
	if !sessionKnown {
		tr.Start(relayer.InstructionProxySyncMarkSessionAsKnown)
		sessionCacheEntry = server.miningSupervisor.MarkSessionAsKnown(
			sessionHeader.SessionId,
			sessionHeader.SessionEndBlockHeight,
		)
		tr.Finish(relayer.InstructionProxySyncMarkSessionAsKnown)
	}

	tr.Start(relayer.InstructionProxySyncCheckSessionIsRewardable)
	sessionIsRewardable := sessionCacheEntry.isRewardable.Load()
	tr.Finish(relayer.InstructionProxySyncCheckSessionIsRewardable)
	if !sessionIsRewardable {
		return relayReq, ErrRelayerProxyRateLimited
	}

	// --- 4) Time budget & write-deadline (helper) ---
	tr.Start(relayer.InstructionProxySyncConfigResponseController)
	requestTimeout := server.requestTimeoutForServiceId(serviceId)
	ctxWithDeadline, cancelDeadline, deadline := setupWriteDeadline(ctx, w, requestTimeout)
	defer cancelDeadline()
	logger = logger.With("deadline", deadline)
	tr.Finish(relayer.InstructionProxySyncConfigResponseController)

	// --- optimistic accounting flags ---
	shouldRewardRelay := false
	isRelayRewardAccumulated := false

	// --- 5) Eager pre-checks (per-service) ---
	isOverServicing := false
	if eagerEnabled {
		// this cleanup will only happen if the relay is in eager validation, since if it is not, all the logic to
		// pay/account will be on the mining supervisor worker.
		defer func() {
			if !shouldRewardRelay && isRelayRewardAccumulated {
				tr.Start(relayer.InstructionProxySyncEagerRewardRollback)
				server.relayMeter.SetNonApplicableRelayReward(ctx, relayReq.Meta)
				tr.Finish(relayer.InstructionProxySyncEagerRewardRollback)
			}
		}()

		tr.Start(relayer.InstructionProxySyncEagerCheckRateLimiting)
		// checking the IsOverServicing implicitly adds the relay to the meter
		isOverServicing = server.relayMeter.IsOverServicing(ctxWithDeadline, meta)
		isRelayRewardAccumulated = true // mark as accumulated for the optimistic accounting
		if isOverServicing && !server.relayMeter.AllowOverServicing() {
			if _, markErr := server.miningSupervisor.MarkSessionAsNonRewardable(sessionHeader.SessionId); markErr != nil {
				logger.Error().Err(markErr).Msgf("❌ Failed marking %s non-rewardable (eager)", sessionHeader.SessionId)
			}
			tr.Finish(relayer.InstructionProxySyncEagerCheckRateLimiting)
			return relayReq, ErrRelayerProxyRateLimited
		}
		tr.Finish(relayer.InstructionProxySyncEagerCheckRateLimiting)

		tr.Start(relayer.InstructionProxySyncEagerRequestVerification)
		verifyErr := server.relayAuthenticator.VerifyRelayRequest(ctxWithDeadline, relayReq, serviceId)
		tr.Finish(relayer.InstructionProxySyncEagerRequestVerification)
		if verifyErr != nil {
			logger.Error().Err(verifyErr).Msg("❌ Failed verifying relay request (eager)")
			return relayReq, verifyErr
		}
	}

	// --- 6) Build backend request ---
	tr.Start(relayer.InstructionProxySyncBuildBackendRequest)
	httpReq, buildErr := relayer.BuildServiceBackendRequest(relayReq, serviceConfig)
	tr.Finish(relayer.InstructionProxySyncBuildBackendRequest)
	if buildErr != nil {
		logger.Error().Err(buildErr).Msg("❌ Failed building backend request")
		return relayReq, ErrRelayerProxyInternalError.Wrapf("build backend request: %v", buildErr)
	}
	relayer.CaptureRequestPreparationDuration(serviceId, start)

	// ensure we still have time
	if err := ctxWithDeadline.Err(); err != nil {
		return relayReq, ErrRelayerProxyTimeout.Wrapf("service %s timed out after %s", serviceId, requestTimeout)
	}

	// derive the remaining budget for the outbound call
	tr.Start(relayer.InstructionProxySyncDeriveRemainingTime)
	ctxRemain, cancelRemain := deriveRemainingBudget(ctxWithDeadline, start, requestTimeout)
	defer cancelRemain()
	httpReq = httpReq.WithContext(ctxRemain)
	tr.Finish(relayer.InstructionProxySyncDeriveRemainingTime)

	// --- 7) Call backend & serialize ---
	tr.Start(relayer.InstructionProxySyncBackendCall)
	serviceCallStart := time.Now()
	httpResp, err := server.httpClient.Do(ctxRemain, logger, httpReq)
	CloseBody(logger, httpReq.Body)
	tr.Finish(relayer.InstructionProxySyncBackendCall)
	if err != nil {
		logger.Error().Err(err).Msg("❌ backend request failed")
		relayer.CaptureServiceDuration(serviceId, serviceCallStart, statusCode)
		if isTimeoutError(err) {
			return relayReq, ErrRelayerProxyTimeout.Wrapf("service %s timed out after %s", serviceId, requestTimeout)
		}
		return relayReq, ErrRelayerProxyInternalError.Wrap(err.Error())
	}
	relayer.CaptureServiceDuration(serviceId, serviceCallStart, httpResp.StatusCode)

	tr.Start(relayer.InstructionProxySyncSerializeBackendResponse)
	wrappedHTTPResponse, responseBz, err := SerializeHTTPResponse(logger, httpResp, server.serverConfig.MaxBodySize)
	CloseBody(logger, httpResp.Body)
	tr.Finish(relayer.InstructionProxySyncSerializeBackendResponse)
	if err != nil {
		logger.Error().Err(err).Msg("serialize service response")
		return relayReq, err
	}
	if httpResp.StatusCode >= http.StatusMultipleChoices {
		logger.Error().
			Int("status_code", httpResp.StatusCode).
			Str("request_url", httpReq.URL.String()).
			Str("request_payload_first_bytes", polylog.Preview(string(relayReq.Payload))).
			Str("response_payload_first_bytes", polylog.Preview(string(wrappedHTTPResponse.BodyBz))).
			Msg("backend non-2XX; passing through")
	}

	// --- 8) Build relay response and write to a client ---
	tr.Start(relayer.InstructionProxySyncGenerateRelayResponse)
	respPrepStart := time.Now()
	relayRes, err := server.newRelayResponse(responseBz, sessionHeader, supplierOperatorAddress)
	tr.Finish(relayer.InstructionProxySyncGenerateRelayResponse)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed building relay response")
		return relayReq, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	relay := &types.Relay{Req: relayReq, Res: relayRes}

	relayer.CaptureResponsePreparationDuration(serviceId, respPrepStart)

	tr.Start(relayer.InstructionProxySyncWriteResponse)
	err = server.sendRelayResponse(relay.Res, w)
	tr.Finish(relayer.InstructionProxySyncWriteResponse)
	if err != nil {
		clientErr := ErrRelayerProxyInternalError.Wrap(err.Error())
		logger.Warn().Err(err).Time("current_time", time.Now()).Msg("❌ write response failed")
		return relayReq, clientErr
	}

	// set ok, since all is good till here and will not modify the response.
	statusCode = http.StatusOK

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msg("relay served successfully")
	relayer.RelaysSuccessTotal.With("service_id", serviceId).Add(1)
	relayer.RelayResponseSizeBytes.With("service_id", serviceId).Observe(float64(relay.Res.Size()))

	// --- 9) Post-write path ---
	if !eagerEnabled {
		// optimistic: publish to supervisor and return
		tr.Start(relayer.InstructionProxySyncPublishToMiningSupervisor)
		if server.miningSupervisor.Publish(ctx, relay) {
			logger.Info().Msg("relay published to mining queue")
		} else {
			logger.Warn().Msg("mining queue saturated/closed; relay not forwarded")
		}
		tr.Finish(relayer.InstructionProxySyncPublishToMiningSupervisor)
		return relayReq, nil
	}

	tr.Start(relayer.InstructionProxySyncEagerCheckRelayEligibility)
	// eager: final eligibility snapshot (session could have expired during backend call)
	rewardEligibilityErr := server.relayAuthenticator.CheckRelayRewardEligibility(ctx, relayReq)
	if rewardEligibilityErr != nil {
		processingMs := time.Since(start).Milliseconds()
		endBlock := server.blockClient.LastBlock(ctx)
		logger.Warn().Msgf(
			"⏱️ Backend took %d ms — relay no longer eligible (session expired: block %d → %d, hash: %X). err: %v",
			processingMs, endBlock.Height()-1, endBlock.Height(), endBlock.Hash(), rewardEligibilityErr,
		)
		isOverServicing = true // treat as over-serviced for reward applicability
	}
	tr.Finish(relayer.InstructionProxySyncEagerCheckRelayEligibility)

	tr.Start(relayer.InstructionProxySyncEagerCheckRewardApplicability)
	// Only mark as rewardable when allowed by service + status + overserve state
	if isRewardApplicable(isOverServicing, httpResp.StatusCode) {
		select {
		case server.servedRewardableRelaysProducer <- relay:
			shouldRewardRelay = true
		default:
			logger.Warn().Msg("mining channel full - dropping (protect tail)")
		}
	}
	tr.Finish(relayer.InstructionProxySyncEagerCheckRewardApplicability)

	return relayReq, nil
}

// parseRelayRequest reads, closes, and basic-validates the request body.
func (server *relayMinerHTTPServer) parseRelayRequest(
	r *http.Request,
	logger polylog.Logger,
) (*types.RelayRequest, error) {
	logger.Debug().Msg("extracting relay request from request body")
	relayReq, err := server.newRelayRequest(r)
	CloseBody(logger, r.Body)
	if err != nil {
		logger.Warn().Err(err).Msg("❌ Failed creating relay request")
		return relayReq, err
	}
	if err := relayReq.ValidateBasic(); err != nil {
		logger.Warn().Err(err).Msg("❌ Failed validating relay request")
		return relayReq, err
	}
	return relayReq, nil
}

// resolveServiceConfig loads the supplier/service config used by this request.
func (server *relayMinerHTTPServer) resolveServiceConfig(
	logger polylog.Logger,
	serviceId string,
	r *http.Request,
) (*config.RelayMinerSupplierServiceConfig, string, error) {
	supplierConfig, ok := server.serverConfig.SupplierConfigsMap[serviceId]
	if !ok {
		return nil, "", ErrRelayerProxyServiceEndpointNotHandled.Wrapf("service %q not configured", serviceId)
	}
	cfg, cfgType, err := getServiceConfig(logger, supplierConfig, r)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed getting service config")
		return nil, "", err
	}
	if cfg == nil {
		return nil, "", ErrRelayerProxyServiceEndpointNotHandled.Wrapf("service %q not configured", serviceId)
	}
	return cfg, cfgType, nil
}

// setupWriteDeadline establishes the overall deadline and applies a write deadline to w.
// returns a context with deadline, its cancel, and the absolute deadline time.
func setupWriteDeadline(
	parent context.Context,
	w http.ResponseWriter,
	requestTimeout time.Duration,
) (context.Context, context.CancelFunc, time.Time) {
	deadline := time.Now().Add(requestTimeout + writeDeadlineSafetyDuration)
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(deadline.Add(writeDeadlineSafetyDuration))
	ctxWithDeadline, cancel := context.WithDeadline(parent, deadline)
	return ctxWithDeadline, cancel, deadline
}

// deriveRemainingBudget clamps the remaining time budget for the backend call.
func deriveRemainingBudget(
	ctxWithDeadline context.Context,
	start time.Time,
	total time.Duration,
) (context.Context, context.CancelFunc) {
	remaining := total - time.Since(start)
	if remaining <= 0 {
		remaining = fallbackTimeout
	}
	return context.WithTimeout(ctxWithDeadline, remaining)
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

//// getSessionEntry checks if the session ID is already known and that all its
//// relevant data is cached to skip late validations or fetching data again.
//func (server *relayMinerHTTPServer) getSessionEntry(sessionId string) bool {
//	//server.knownSessionsMutex.RLock()
//	//defer server.knownSessionsMutex.RUnlock()
//	//_, ok := server.knownSessions[sessionId]
//	_, ok := server.knownSessions.Load(sessionId)
//	return ok
//}
//
//// markSessionAsKnown marks the session ID as known to avoid late validations
//// for subsequent requests within the same session.
//func (server *relayMinerHTTPServer) markSessionAsKnown(sessionId string, sessionEndBlockHeight int64) {
//	//server.knownSessionsMutex.Lock()
//	//defer server.knownSessionsMutex.Unlock()
//	//server.knownSessions[sessionId] = sessionEndBlockHeight
//	server.knownSessions.Store(sessionId, sessionEndBlockHeight)
//}
//
//// pruneOutdatedKnownSessions removes known sessions that have ended before the
//// current block height to free up memory and keep the known sessions map up-to-date.
//func (server *relayMinerHTTPServer) pruneOutdatedKnownSessions(ctx context.Context, block client.Block) {
//	// TODO_IMPROVE(@red-0ne): Do not prune at each block, instead do it periodically each num blocks per session.
//	//server.knownSessionsMutex.Lock()
//	//defer server.knownSessionsMutex.Unlock()
//	//
//	//for sessionId, endHeight := range server.knownSessions {
//	//	// TODO_IMPROVE(@red-0ne):
//	//	// 1. Replace (endHeight+1) with (endHeight + gracePeriod) to avoid prematurely pruning sessions of late requests
//	//	// 2. Only prune when (current_height > endHeight + gracePeriod), ensuring the session is definitively out of service.
//	//	if endHeight+1 < block.Height() {
//	//		delete(server.knownSessions, sessionId)
//	//	}
//	//}
//	server.knownSessions.Range(func(sessionId string, endHeight int64) bool {
//		if endHeight+1 < block.Height() {
//			server.knownSessions.Delete(sessionId)
//		}
//		return true
//	})
//}

// isTimeoutError checks if the error is a timeout error.
// It is used to determine if the request timed out by verified if
// the error is a context deadline exceeded error.
func isTimeoutError(err error) bool {
	urlErr, ok := err.(*url.Error)
	return ok && urlErr.Timeout()
}
