package proxy

import (
	"context"

	sessiontypes "github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest is a shared method used by RelayServers to check the relay request signature and session validity.
func (rp *relayerProxy) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *types.RelayRequest,
	service *sharedtypes.Service,
) error {
	if err := rp.ringCache.VerifyRelayRequestSignature(ctx, relayRequest); err != nil {
		return err
	}

	// Application address is used to verify the relayRequest signature, it is
	// guaranteed to be present in the relayRequest since the signature has already
	// been verified.
	appAddress := relayRequest.GetMeta().GetSessionHeader().GetApplicationAddress()

	// Query for the current session to check if relayRequest sessionId matches the current session.
	rp.logger.Debug().
		Fields(map[string]any{
			"session_id":          relayRequest.GetMeta().GetSessionHeader().GetSessionId(),
			"application_address": relayRequest.GetMeta().GetSessionHeader().GetApplicationAddress(),
			"service_id":          relayRequest.GetMeta().GetSessionHeader().GetService().GetId(),
		}).
		Msg("verifying relay request session")

	sessionBlockHeight, err := rp.getTargetSessionBlockHeight(ctx, relayRequest)
	if err != nil {
		return err
	}

	session, err := rp.sessionQuerier.GetSession(
		ctx,
		appAddress,
		service.Id,
		sessionBlockHeight,
	)

	if err != nil {
		return err
	}

	// Since the retrieved sessionId was in terms of:
	// - the current block height and sessionGracePeriod (which are not provided by the relayRequest)
	// - serviceId (which is not provided by the relayRequest)
	// - applicationAddress (which is used to to verify the relayRequest signature)
	// we can reduce the session validity check to checking if the retrieved session's sessionId
	// matches the relayRequest sessionId.
	// TODO_INVESTIGATE: Revisit the assumptions above at some point in the future, but good enough for now.
	if session.SessionId != relayRequest.GetMeta().GetSessionHeader().GetSessionId() {
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

	// Check if the `RelayRequest`'s session has expired.
	if sessionEndblockHeight < currentBlockHeight {
		// Do not process the `RelayRequest` if the session has expired and the current
		// block height is outside the session's grace period.
		if sessiontypes.IsWithinGracePeriod(sessionEndblockHeight, currentBlockHeight) {
			return sessionEndblockHeight, nil
		}

		return 0, ErrRelayerProxyInvalidSession.Wrapf(
			"session expired, expecting: %d, got: %d",
			sessionEndblockHeight,
			currentBlockHeight,
		)
	}

	return currentBlockHeight, nil
}
