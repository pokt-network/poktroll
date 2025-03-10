package relay_authenticator

import (
	"context"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
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
	if err = ra.ringCache.VerifyRelayRequestSignature(ctx, relayRequest); err != nil {
		return err
	}

	meta := relayRequest.GetMeta()

	// Extract the session header for usage below.
	// ringCache.VerifyRelayRequestSignature already verified the header's validity.
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
			"session ID mismatch, expecting: %+v, got: %+v",
			session.GetSessionId(),
			relayRequest.Meta.GetSessionHeader().GetSessionId(),
		)
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	_, isSupplierOperatorAddressPresent := ra.operatorAddressToSigningKeyNameMap[meta.GetSupplierOperatorAddress()]
	if !isSupplierOperatorAddressPresent {
		return ErrRelayAuthenticatorMissingSupplierOperatorAddress
	}

	for _, supplier := range session.Suppliers {
		// Verify if the supplier operator address in the session matches the one in the relayRequest.
		if supplier.OperatorAddress == meta.GetSupplierOperatorAddress() {
			return nil
		}
	}

	return ErrRelayAuthenticatorInvalidSessionSupplier
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
	currentHeight := ra.blockClient.LastBlock(ctx).Height()
	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	sharedParams, err := ra.sharedQuerier.GetParams(ctx)
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
			"session expired, expecting: %d, got: %d",
			sessionEndHeight,
			currentHeight,
		)
	}

	// The RelayRequest's session is active, return the current block height.
	return currentHeight, nil
}
