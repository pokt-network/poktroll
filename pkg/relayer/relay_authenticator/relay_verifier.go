package relay_authenticator

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest checks the relay request signature and session validity.
func (ra *relayAuthenticator) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
	supplierServiceId string,
) error {
	// Get the block height at which the relayRequest should be processed.
	// Check if the relayRequest is on time or within the session's grace period
	// before attempting to verify the relayRequest signature.
	sessionBlockHeight, err := ra.getTargetSessionBlockHeight(ctx, relayRequest)
	if err != nil {
		return err
	}

	// Verify the relayRequest metadata, signature, session header and other
	// basic validation.
	if err = ra.ringClient.VerifyRelayRequestSignature(ctx, relayRequest); err != nil {
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
	session, err := ra.sessionQuerier.GetSession(
		ctx,
		appAddress,
		supplierServiceId,
		sessionBlockHeight,
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
	if session.SessionId != sessionHeader.GetSessionId() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session ID mismatch, expecting: %s, got: %s. "+
				"This may indicate a full node synchronization issue. "+
				"Please verify your full node is in sync and not overwhelmed with websocket connections.",
			session.GetSessionId(),
			relayRequest.Meta.GetSessionHeader().GetSessionId(),
		)
	}

	// A matching session ID does NOT validate the header's OWN fields.
	//
	// The session ID is sha3(blockHash | serviceId | appAddress | sessionStartHeight)
	// (x/session/keeper/session_hydrator.go GetSessionId), so a matching ID only proves
	// the client KNEW those values -- every field of the header is still whatever the
	// client put in it. The only other constraint is SessionHeader.ValidateBasic
	// (start >= 1, end > start), so a request may pair a legitimate session ID with an
	// arbitrary session_start_block_height (or service ID, or end height).
	//
	// Left unchecked, that mismatch is fatal LATER, onchain, and the SUPPLIER pays:
	//
	//   - The relay is filed under (supplier, session END height, session ID)
	//     (relayer/session/session.go ensureSessionTree), all of which the forged header
	//     can set correctly, so it lands in the SAME tree as the session's honest relays.
	//   - At proof time the chain re-compares the SAMPLED relay's header field-by-field
	//     against the claim's (x/proof/keeper/proof_validation.go compareSessionHeaders,
	//     which checks start and end height explicitly). Sampling a forged relay yields
	//     ErrProofInvalidRelay -- an invalid proof, and the supplier is SLASHED.
	//   - The forged start height is also the height the miner resolves relay mining
	//     difficulty at (relayer/miner/miner.go) and the height that weights the relay
	//     (relayer/session/service.go). A stale height resolves an easier target than the
	//     chain validates against, and a different compute-units-per-relay than the chain
	//     prices the claim with.
	//
	// So enforce here exactly the equality the chain enforces at proof time. The onchain
	// session was already fetched above, so this costs NO additional query.
	if err = compareSessionHeaders(session.GetHeader(), sessionHeader); err != nil {
		return err
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

// compareSessionHeaders returns an error if any field of the relay request's session
// header differs from the corresponding field of the onchain session's header.
//
// It deliberately mirrors x/proof/keeper.compareSessionHeaders, which the chain applies
// to the sampled relay during proof validation. Enforcing the same equality at the door
// means a relay that would invalidate the claim (and slash the supplier) is rejected
// instead of being mined into the session tree.
//
// The session ID is compared last: its dedicated check upstream carries an operator-facing
// "is your full node in sync" hint, so a mismatch there is expected to be caught before
// reaching this function.
func compareSessionHeaders(onchainSessionHeader, requestSessionHeader *sessiontypes.SessionHeader) error {
	if requestSessionHeader.GetApplicationAddress() != onchainSessionHeader.GetApplicationAddress() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session header application address mismatch, expecting: %q, got: %q",
			onchainSessionHeader.GetApplicationAddress(),
			requestSessionHeader.GetApplicationAddress(),
		)
	}

	if requestSessionHeader.GetServiceId() != onchainSessionHeader.GetServiceId() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session header service ID mismatch, expecting: %q, got: %q",
			onchainSessionHeader.GetServiceId(),
			requestSessionHeader.GetServiceId(),
		)
	}

	if requestSessionHeader.GetSessionStartBlockHeight() != onchainSessionHeader.GetSessionStartBlockHeight() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session header session start height mismatch, expecting: %d, got: %d",
			onchainSessionHeader.GetSessionStartBlockHeight(),
			requestSessionHeader.GetSessionStartBlockHeight(),
		)
	}

	if requestSessionHeader.GetSessionEndBlockHeight() != onchainSessionHeader.GetSessionEndBlockHeight() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session header session end height mismatch, expecting: %d, got: %d",
			onchainSessionHeader.GetSessionEndBlockHeight(),
			requestSessionHeader.GetSessionEndBlockHeight(),
		)
	}

	if requestSessionHeader.GetSessionId() != onchainSessionHeader.GetSessionId() {
		return ErrRelayAuthenticatorInvalidSession.Wrapf(
			"session header session ID mismatch, expecting: %q, got: %q",
			onchainSessionHeader.GetSessionId(),
			requestSessionHeader.GetSessionId(),
		)
	}

	return nil
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

	// Resolve the window under the params epoch effective at THIS session's end height,
	// matching x/proof validateClaimWindow. Measuring an old-epoch session against live
	// window offsets rejects relays the chain would still have rewarded (or accepts relays
	// past the real cutoff, which are then unpaid).
	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()
	sharedParams, err := ra.sharedQuerier.GetParamsAtHeight(ctx, sessionEndHeight)
	if err != nil {
		return err
	}

	sessionClaimOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

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
	currentBlock := ra.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()

	ra.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) during session validation",
		currentHeight,
		currentBlock.Hash(),
	)
	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	// grace_period_end_offset_blocks is session TIMING, so it resolves at the session
	// END height, mirroring the chain (x/session's session hydrator and x/proof both
	// resolve it at-height). CheckRelayRewardEligibility in this same file was already
	// converted; this path was missed. Reading live params here would reject relays the
	// chain still counts in the session after governance shortens the grace period, and
	// accept relays past the chain's real cutoff -- served unpaid -- after it lengthens.
	sharedParams, err := ra.sharedQuerier.GetParamsAtHeight(ctx, sessionEndHeight)
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
