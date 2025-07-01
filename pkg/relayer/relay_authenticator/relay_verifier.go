package relay_authenticator

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest checks the relay request signature and session validity.
func (ra *relayAuthenticator) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) error {
	// Get the block height at which the relayRequest should be processed.
	// Check if the relayRequest is on time or within the session's grace period
	// before attempting to verify the relayRequest signature.
	relayProcessingBlockHeight, err := ra.getRelayProcessingBlockHeight(ctx, relayRequest)
	if err != nil {
		return err
	}

	// Verify the relayRequest metadata, signature, session header and other
	// basic validation.
	if err = ra.ringClient.VerifyRelayRequestSignature(ctx, relayRequest); err != nil {
		return err
	}

	relayMeta := relayRequest.GetMeta()
	// Extract the session header for usage below.
	// ringClient.VerifyRelayRequestSignature already verified the header's validity.
	relaySessionHeader := relayMeta.SessionHeader
	relayServiceId := relaySessionHeader.GetServiceId()
	// Application address is used to verify the relayRequest signature.
	// It is guaranteed to be present in the relayRequest since the signature
	// has already been verified.
	relayAppAddress := relaySessionHeader.GetApplicationAddress()

	ra.logger.Debug().
		Fields(map[string]any{
			"session_id":                relaySessionHeader.GetSessionId(),
			"application_address":       relayAppAddress,
			"service_id":                relayServiceId,
			"supplier_operator_address": relayMeta.GetSupplierOperatorAddress(),
		}).
		Msg("verifying relay request session")

	// Query for the current session to check if relayRequest sessionId matches the current session.
	session, err := ra.sessionQuerier.GetSession(
		ctx,
		relayAppAddress,
		relayServiceId,
		relayProcessingBlockHeight,
	)
	if err != nil {
		return err
	}

	// Session validity can be checked via a basic ID comparison due to the reasons below.
	//
	// Since the retrieved sessionId was in terms of:
	// - the current block height and sessionGracePeriod (which are not provided by the relayRequest)
	// - serviceId (which is not provided by the relayRequest)
	// - applicationAddress (which is used to verify the relayRequest signature)
	if session.SessionId != relaySessionHeader.GetSessionId() {
		logSessionIDMismatch(ra.logger, session, relaySessionHeader, &relayMeta)

		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"Session ID mismatch: expected %s, got %s (expected block range: [%d-%d], got: [%d-%d]). See logs for details.",
			session.SessionId, relaySessionHeader.GetSessionId(),
			session.Header.SessionStartBlockHeight, session.Header.SessionEndBlockHeight,
			relayMeta.SessionHeader.SessionStartBlockHeight, relayMeta.SessionHeader.SessionEndBlockHeight,
		)
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	_, isSupplierOperatorAddressPresent := ra.operatorAddressToSigningKeyNameMap[relayMeta.GetSupplierOperatorAddress()]
	if !isSupplierOperatorAddressPresent {
		return ErrRelayAuthenticatorMissingSupplierOperatorAddress.Wrapf(
			"supplier operator address %s is not present in the signing key names map",
			relayMeta.GetSupplierOperatorAddress(),
		)
	}

	for _, supplier := range session.Suppliers {

		// Verify if the supplier operator address in the session matches the one in the relayRequest.
		if supplier.OperatorAddress == relayMeta.GetSupplierOperatorAddress() {
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
	relaySessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) during reward eligibility check",
		currentHeight,
		currentBlock.Hash(),
	)

	sharedParams, err := ra.sharedQuerier.GetParams(ctx)
	if err != nil {
		return err
	}

	sessionGracePeriodEndHeight := sharedtypes.GetSessionGracePeriodEndHeight(
		sharedParams,
		relaySessionEndHeight,
	)

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"⏳ Checking relay reward eligibility. Checking if the current height (%d) can process relay with session end height (%d) before the grace period ends at height (%d)",
		currentHeight,
		relaySessionEndHeight,
		sessionGracePeriodEndHeight,
	)

	// If current height is equal or greater than the grace period end height,
	// the relay is no longer eligible for rewards as the session has expired for reward purposes.
	if currentHeight >= sessionGracePeriodEndHeight {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"(⌛) SESSION EXPIRED! Relay block height (%d) is past the session end block height (%d) AND the grace period has elapsed. Make sure that your both your full node and the Gateway's full node are in sync. ",
			sessionGracePeriodEndHeight,
			currentHeight,
		)
	}

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"✅ Relay is eligible for rewards - current height (%d) < session grace period end height (%d)",
		currentHeight,
		sessionGracePeriodEndHeight,
	)

	return nil
}

// getRelayProcessingBlockHeight returns the block height at which the session
// for the given relayRequest should be processed.
//   - If the request is within the session bounds, the current block height is returned.
//   - If the request is outside the session bounds, but within the session's grace period, the session's end block height is returned.
//   - In all other cases, an error is returned.
func (ra *relayAuthenticator) getRelayProcessingBlockHeight(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) (relayProcessingBlockHeight int64, err error) {
	currentBlock := ra.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) during session validation",
		currentHeight,
		currentBlock.Hash(),
	)
	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	// If the session end height is greater than or equal to the current height,
	// the session is still active and the current block height should be used.
	if sessionEndHeight >= currentHeight {
		return currentHeight, nil
	}

	// The session has ended so we need to validate if the session is still within the grace period.
	// If the session is within the grace period, the session's end block height should be used.
	// Otherwise, the session is invalid and an error should be returned.

	// Retrieve the shared parameters to check if the session is within the grace period.
	sharedParams, err := ra.sharedQuerier.GetParams(ctx)
	if err != nil {
		return -1, err
	}

	// Do not process the `RelayRequest` if the session has expired and the current
	// block height is outside the session's grace period.
	isSessionValid := !sharedtypes.IsGracePeriodElapsed(sharedParams, sessionEndHeight, currentHeight)
	if !isSessionValid {
		// The RelayRequest's session has expired but is still within the
		// grace period, process it as if the session is still active.
		return sessionEndHeight, nil
	}

	return -1, ErrRelayAuthenticatorInvalidSession.Wrapf(
		"(⌛) SESSION EXPIRED! Relay block height (%d) is past the session end block height (%d) AND the grace period has elapsed. Make sure that your both your full node and the Gateway's full node are in sync. ",
		sessionEndHeight,
		currentHeight,
	)
}

// logSessionIDMismatch logs detailed information about a session ID mismatch during relay verification.
func logSessionIDMismatch(
	logger polylog.Logger,
	session *sessiontypes.Session,
	relaySessionHeader *sessiontypes.SessionHeader,
	relayMeta *servicetypes.RelayRequestMetadata,
) {
	expectedSessionID := session.SessionId
	receivedSessionID := relaySessionHeader.GetSessionId()
	expectedStart := session.Header.SessionStartBlockHeight
	expectedEnd := session.Header.SessionEndBlockHeight
	receivedStart := relayMeta.SessionHeader.SessionStartBlockHeight
	receivedEnd := relayMeta.SessionHeader.SessionEndBlockHeight

	// Determine if we're ahead or behind
	var syncStatus, emoji string
	if receivedEnd < expectedStart {
		syncStatus = "RelayMiner is BEHIND 🐢"
		emoji = "🐢"
	} else if receivedStart > expectedEnd {
		syncStatus = "RelayMiner is AHEAD 🚀"
		emoji = "🚀"
	} else {
		syncStatus = "RelayMiner is OUT OF SYNC ⚠️"
		emoji = "⚠️"
	}

	logger.Error().
		Str("error", "Session ID Mismatch").
		Msg(fmt.Sprintf(
			"Session ID mismatch detected while verifying relay request.\n"+
				"Expected session_id: %s, got: %s.\n"+
				"Expected block range: [%d-%d], got: [%d-%d].\n"+
				"%s %s\n"+
				"Please verify your full node is in sync or not overwhelmed with websocket connections.",
			expectedSessionID, receivedSessionID,
			expectedStart, expectedEnd,
			receivedStart, receivedEnd,
			emoji, syncStatus,
		))
}
