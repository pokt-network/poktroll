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
		sessionHeader := session.Header
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"Session ID mismatch: expected %s, got %s (expected block range: [%d-%d], got: [%d-%d]). See logs for details.",
			session.SessionId, relaySessionHeader.GetSessionId(),
			sessionHeader.SessionStartBlockHeight, sessionHeader.SessionEndBlockHeight,
			relaySessionHeader.SessionStartBlockHeight, relaySessionHeader.SessionEndBlockHeight,
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

// CheckRelayRewardEligibility verifies if a relay is still eligible for rewards.
//
// A relay is eligible for rewards if it's processed before the grace period ends.
// This ensures that relays arriving late due to network latency can still be rewarded,
// but prevents indefinite reward claims for old sessions.
//
// Timeline: [Session End] -> [Grace Period] -> [Claim Window] -> [Proof Window]
//
//	^^^^^^^^^^^^^^^
//	Relays must be processed here to earn rewards
//
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

	gracePeriodEndHeight := sharedtypes.GetSessionGracePeriodEndHeight(
		sharedParams,
		relaySessionEndHeight,
	)

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"⏳ Checking relay reward eligibility: current height (%d), session end (%d), grace period end (%d)",
		currentHeight,
		relaySessionEndHeight,
		gracePeriodEndHeight,
	)

	// Check if grace period has elapsed (relay is no longer eligible for rewards)
	if !isRelayWithinGracePeriod(sharedParams, relaySessionEndHeight, currentHeight) {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"(⌛) REWARD ELIGIBILITY EXPIRED! Current height (%d) is past the grace period end height (%d) "+
				"for session ending at %d. Grace period: %d blocks. "+
				"Ensure both your full node and the Gateway's full node are in sync.",
			currentHeight,
			gracePeriodEndHeight,
			relaySessionEndHeight,
			sharedParams.GetGracePeriodEndOffsetBlocks(),
		)
	}

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"✅ Relay is eligible for rewards - relay session end height (%d) is within grace period (ends at height %d)",
		relaySessionEndHeight,
		gracePeriodEndHeight,
	)

	return nil
}

// getRelayProcessingBlockHeight returns the block height at which the session
// for the given relayRequest should be processed.
//
// The function determines the appropriate block height based on the relay timing:
//   - If the relay arrives during the active session: returns current block height
//   - If the relay arrives during grace period: returns session end height
//     (This allows late relays to be processed as if they arrived at session end)
//   - If the relay arrives after grace period: returns error (relay is too late)
//
// Grace period provides a buffer after session end to accommodate network latency
// and clock differences between nodes.
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

	// Case 1: Relay arrived during active session - process at current height
	if sessionEndHeight >= currentHeight {
		return currentHeight, nil
	}

	// Case 2: Session has ended - check if we're within grace period
	sharedParams, err := ra.sharedQuerier.GetParams(ctx)
	if err != nil {
		return -1, err
	}

	if isRelayWithinGracePeriod(sharedParams, sessionEndHeight, currentHeight) {
		// Case 2a: Within grace period - process as if relay arrived at session end
		// This ensures consistent session assignment for late-arriving relays
		return sessionEndHeight, nil
	}

	// Case 3: Grace period has elapsed - relay is too late
	gracePeriodEndHeight := sharedtypes.GetSessionGracePeriodEndHeight(sharedParams, sessionEndHeight)
	return -1, ErrRelayAuthenticatorInvalidSession.Wrapf(
		"(⌛) SESSION EXPIRED! Current height (%d) is past the grace period end height (%d) for session ending at %d. "+
			"Grace period: %d blocks. Make sure that both your full node and the Gateway's full node are in sync.",
		currentHeight,
		gracePeriodEndHeight,
		sessionEndHeight,
		sharedParams.GetGracePeriodEndOffsetBlocks(),
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

// isRelayWithinGracePeriod checks if a relay is within the grace period for a given session.
// Returns true if the relay can still be processed, false if the grace period has elapsed.
func isRelayWithinGracePeriod(
	sharedParams *sharedtypes.Params,
	sessionEndHeight int64,
	currentHeight int64,
) bool {
	gracePeriodEndHeight := sharedtypes.GetSessionGracePeriodEndHeight(sharedParams, sessionEndHeight)
	return currentHeight < gracePeriodEndHeight
}
