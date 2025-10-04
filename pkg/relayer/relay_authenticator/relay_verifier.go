package relay_authenticator

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest checks the relay request signature and session validity.
func (ra *relayAuthenticator) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
	supplierServiceId string,
) error {
	// One buffered tracker per processed relay; flush at end.
	ctxPerf, tr, flush := relayer.EnsurePerfBuffered(ctx, "")
	defer flush()
	// Get the block height at which the relayRequest should be processed.
	// Check if the relayRequest is on time or within the session's grace period
	// before attempting to verify the relayRequest signature.
	tr.Start("relay_authenticator_get_target_session_block_height")
	sessionBlockHeight, err := ra.getTargetSessionBlockHeight(ctxPerf, relayRequest)
	tr.Finish("relay_authenticator_get_target_session_block_height")
	if err != nil {
		return err
	}

	// Verify the relayRequest metadata, signature, session header and other
	// basic validation.
	tr.Start("relay_authenticator_verify_relay_request_signature")
	err = ra.ringClient.VerifyRelayRequestSignature(ctx, relayRequest)
	tr.Finish("relay_authenticator_verify_relay_request_signature")
	if err != nil {
		return err
	}

	meta := relayRequest.GetMeta()

	// Extract the session header for usage below.
	// ringClient.VerifyRelayRequestSignature already verified the header's validity.
	sessionHeader := meta.SessionHeader

	// Application address is used to verify the relayRequest signature.
	// It is guaranteed to be present in the relayRequest since the signature
	// has already been verified.
	appAddress := sessionHeader.GetApplicationAddress()

	ra.logger.Debug().
		Fields(map[string]any{
			"session_id":                sessionHeader.GetSessionId(),
			"application_address":       appAddress,
			"service_id":                sessionHeader.GetServiceId(),
			"supplier_operator_address": meta.GetSupplierOperatorAddress(),
		}).
		Msg("verifying relay request session")

	// Query for the current session to check if relayRequest sessionId matches the current session.
	tr.Start("relay_authenticator_get_session")
	session, err := ra.sessionQuerier.GetSession(
		ctx,
		appAddress,
		supplierServiceId,
		sessionBlockHeight,
	)
	tr.Finish("relay_authenticator_get_session")
	if err != nil {
		return err
	}

	// Session validity can be checked via a basic ID comparison due to the reasons below.
	//
	// Since the retrieved sessionId was in terms of:
	// - the current block height and sessionGracePeriod (which are not provided by the relayRequest)
	// - serviceId (which is not provided by the relayRequest)
	// - applicationAddress (which is used to verify the relayRequest signature)
	if session.SessionId != sessionHeader.GetSessionId() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session ID mismatch, expecting: %s, got: %s. "+
				"This may indicate a full node synchronization issue. "+
				"Please verify your full node is in sync and not overwhelmed with websocket connections.",
			session.GetSessionId(),
			relayRequest.Meta.GetSessionHeader().GetSessionId(),
		)
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	_, isSupplierOperatorAddressPresent := ra.operatorAddressToSigningKeyNameMap[meta.GetSupplierOperatorAddress()]
	if !isSupplierOperatorAddressPresent {
		return ErrRelayAuthenticatorMissingSupplierOperatorAddress.Wrapf(
			"supplier operator address %s is not present in the signing key names map",
			meta.GetSupplierOperatorAddress(),
		)
	}

	for _, supplier := range session.Suppliers {
		// Verify if the supplier operator address in the session matches the one in the relayRequest.
		if supplier.OperatorAddress == meta.GetSupplierOperatorAddress() {
			return nil
		}
	}

	return ErrRelayAuthenticatorInvalidSessionSupplier
}

// CheckRelayRewardEligibility verifies the relay's session hasn't expired for reward
// purposes by ensuring the current block height hasn't reached the claim window yet.
// Returns an error if the relay is no longer eligible for rewards.
func (ra *relayAuthenticator) CheckRelayRewardEligibility(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) error {
	currentBlock := ra.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) during reward eligibility check",
		currentHeight,
		currentBlock.Hash(),
	)

	sharedParams, err := ra.sharedQuerier.GetParams(ctx)
	if err != nil {
		return err
	}

	sessionClaimOpenHeight := sharedtypes.GetClaimWindowOpenHeight(
		sharedParams,
		relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight(),
	)

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"⏳ Checking relay reward eligibility - relay must be processed before claim window opens at height %d",
		sessionClaimOpenHeight,
	)

	// If current height is equal or greater than the claim window opening height,
	// the relay is no longer eligible for rewards as the session has expired
	// for reward purposes
	if currentHeight >= sessionClaimOpenHeight {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session expired, must be before claim window open height (%d), but current height is (%d). "+
				"This may indicate a full node synchronization issue. "+
				"Please verify your full node is in sync and not overwhelmed with websocket connections.",
			sessionClaimOpenHeight,
			currentHeight,
		)
	}

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"✅ Relay is eligible for rewards - current height (%d) < claim window open height (%d)",
		currentHeight,
		sessionClaimOpenHeight,
	)

	return nil
}

// getTargetSessionBlockHeight returns the block height at which the session
// for the given relayRequest should be processed.
//   - If the session is within the grace period, the session's end block height is returned.
//   - Otherwise, the current block height is returned.
//   - If the session has expired, then return an error.
func (ra *relayAuthenticator) getTargetSessionBlockHeight(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) (sessionHeight int64, err error) {
	// One buffered tracker per processed relay; flush at end.
	ctxPerf, tr, flush := relayer.EnsurePerfBuffered(ctx, "")
	defer flush()

	tr.Start("relay_authenticator_get_current_block_height")
	currentBlock := ra.blockClient.LastBlock(ctxPerf)
	tr.Finish("relay_authenticator_get_current_block_height")
	currentHeight := currentBlock.Height()

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) during session validation",
		currentHeight,
		currentBlock.Hash(),
	)
	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	tr.Start("relay_authenticator_get_params")
	sharedParams, err := ra.sharedQuerier.GetParams(ctxPerf)
	tr.Finish("relay_authenticator_get_params")
	if err != nil {
		return 0, err
	}

	// Check if the RelayRequest's session has expired.
	if sessionEndHeight < currentHeight {
		// Do not process the `RelayRequest` if the session has expired and the current
		// block height is outside the session's grace period.
		if !sharedtypes.IsGracePeriodElapsed(sharedParams, sessionEndHeight, currentHeight) {
			// The RelayRequest's session has expired but is still within the
			// grace period, process it as if the session is still active.
			return sessionEndHeight, nil
		}

		return 0, ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session expired, expecting: %d, got: %d. "+
				"This may indicate network delay or RelayMiner overload.",
			sessionEndHeight,
			currentHeight,
		)
	}

	// The RelayRequest's session is active, return the current block height.
	return currentHeight, nil
}
