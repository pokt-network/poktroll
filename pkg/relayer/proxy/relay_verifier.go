package proxy

import (
	"context"

	sessiontypes "github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest is a shared method used by RelayServers to check the relay
// request signature and session validity.
func (rp *relayerProxy) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *types.RelayRequest,
	supplierService *sharedtypes.Service,
) error {
	// Verify the relayRequest metadata, signature, session header and other
	// basic validation.
	if err := rp.ringCache.VerifyRelayRequestSignature(ctx, relayRequest); err != nil {
		return err
	}

	// Extract the session header for usage below.
	// ringCache.VerifyRelayRequestSignature already verified the header's validaity.
	sessionHeader := relayRequest.GetMeta().SessionHeader

	// Application address is used to verify the relayRequest signature.
	// It is guaranteed to be present in the relayRequest since the signature
	// has already been verified.
	appAddress := sessionHeader.GetApplicationAddress()

	rp.logger.Debug().
		Fields(map[string]any{
			"session_id":          sessionHeader.GetSessionId(),
			"application_address": appAddress,
			"service_id":          sessionHeader.GetService().GetId(),
		}).
		Msg("verifying relay request session")

	// Get the block height at which the relayRequest should be processed.
	sessionBlockHeight, err := rp.getTargetSessionBlockHeight(ctx, relayRequest)
	if err != nil {
		return err
	}

	// Query for the current session to check if relayRequest sessionId matches the current session.
	session, err := rp.sessionQuerier.GetSession(
		ctx,
		appAddress,
		supplierService.Id,
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
	// - applicationAddress (which is used to to verify the relayRequest signature)
	//
	// TODO_BLOCKER: Revisit the assumptions above but good enough for now.
	if session.SessionId != sessionHeader.GetSessionId() {
		return ErrRelayerProxyInvalidSession.Wrapf(
			"session mismatch, expecting: %+v, got: %+v",
			session.Header,
			relayRequest.Meta.SessionHeader,
		)
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	for _, supplier := range session.Suppliers {
		if supplier.Address == rp.supplierAddress {
			return nil
		}
	}

	return ErrRelayerProxyInvalidSupplier
}

// getTargetSessionBlockHeight returns the block height at which the session
// for the given relayRequest should be processed. If the session is within the
// grace period, the session's end block height is returned. Otherwise,
// the current block height is returned.
// If the session has expired, then return an error.
func (rp *relayerProxy) getTargetSessionBlockHeight(
	ctx context.Context,
	relayRequest *types.RelayRequest,
) (sessionBlockHeight int64, err error) {
	currentBlockHeight := rp.blockClient.LastNBlocks(ctx, 1)[0].Height()
	sessionEndblockHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	// Check if the RelayRequest's session has expired.
	if sessionEndblockHeight < currentBlockHeight {
		// Do not process the `RelayRequest` if the session has expired and the current
		// block height is outside the session's grace period.
		if sessiontypes.IsWithinGracePeriod(sessionEndblockHeight, currentBlockHeight) {
			// The RelayRequest's session has expired but is still within the
			// grace period so process it as if the session is still active.
			return sessionEndblockHeight, nil
		}

		return 0, ErrRelayerProxyInvalidSession.Wrapf(
			"session expired, expecting: %d, got: %d",
			sessionEndblockHeight,
			currentBlockHeight,
		)
	}

	// The RelayRequest's session is active so return the current block height.
	return currentBlockHeight, nil
}
