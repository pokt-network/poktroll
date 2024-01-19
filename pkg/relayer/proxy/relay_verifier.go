package proxy

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ring "github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest is a shared method used by RelayServers to check the relay request signature and session validity.
func (rp *relayerProxy) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *types.RelayRequest,
	service *sharedtypes.Service,
) error {
	rp.logger.Debug().
		Fields(map[string]any{
			"session_id":          relayRequest.Meta.SessionHeader.SessionId,
			"application_address": relayRequest.Meta.SessionHeader.ApplicationAddress,
			"service_id":          relayRequest.Meta.SessionHeader.Service.Id,
		}).
		Msg("verifying relay request signature")

	// extract the relay request's ring signature
	if relayRequest.Meta == nil {
		return ErrRelayerProxyEmptyRelayRequestSignature.Wrapf(
			"request payload: %s", relayRequest.Payload,
		)
	}
	signature := relayRequest.Meta.Signature
	if signature == nil {
		return sdkerrors.Wrapf(
			ErrRelayerProxyInvalidRelayRequest,
			"missing signature from relay request: %v", relayRequest,
		)
	}

	ringSig := new(ring.RingSig)
	if err := ringSig.Deserialize(ring_secp256k1.NewCurve(), signature); err != nil {
		return sdkerrors.Wrapf(
			ErrRelayerProxyInvalidRelayRequestSignature,
			"error deserializing ring signature: %v", err,
		)
	}

	if relayRequest.Meta.SessionHeader.ApplicationAddress == "" {
		return sdkerrors.Wrap(
			ErrRelayerProxyInvalidRelayRequest,
			"missing application address from relay request",
		)
	}

	// get the ring for the application address of the relay request
	appAddress := relayRequest.Meta.SessionHeader.ApplicationAddress
	appRing, err := rp.ringCache.GetRingForAddress(ctx, appAddress)
	if err != nil {
		return sdkerrors.Wrapf(
			ErrRelayerProxyInvalidRelayRequest,
			"error getting ring for application address %s: %v", appAddress, err,
		)
	}

	// verify the ring signature against the ring
	if !ringSig.Ring().Equals(appRing) {
		return sdkerrors.Wrapf(
			ErrRelayerProxyInvalidRelayRequestSignature,
			"ring signature does not match ring for application address %s", appAddress,
		)
	}

	// get and hash the signable bytes of the relay request
	requestSignableBz, err := relayRequest.GetSignableBytesHash()
	if err != nil {
		return sdkerrors.Wrapf(ErrRelayerProxyInvalidRelayRequest, "error getting signable bytes: %v", err)
	}

	// verify the relay request's signature
	if valid := ringSig.Verify(requestSignableBz); !valid {
		return sdkerrors.Wrapf(
			ErrRelayerProxyInvalidRelayRequestSignature,
			"invalid ring signature",
		)
	}

	// Query for the current session to check if relayRequest sessionId matches the current session.
	rp.logger.Debug().
		Fields(map[string]any{
			"session_id":          relayRequest.Meta.SessionHeader.SessionId,
			"application_address": relayRequest.Meta.SessionHeader.ApplicationAddress,
			"service_id":          relayRequest.Meta.SessionHeader.Service.Id,
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
	if session.SessionId != relayRequest.Meta.SessionHeader.SessionId {
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
// for the given relayRequest should be processed. If the session is in the
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
		// block height is past the session's grace period.
		if sessionEndblockHeight < currentBlockHeight-sessionkeeper.SessionGracePeriod {
			return 0, ErrRelayerProxyInvalidSession.Wrapf(
				"session expired, expecting: %d, got: %d",
				sessionEndblockHeight,
				currentBlockHeight,
			)
		}

		return sessionEndblockHeight, nil
	}

	return currentBlockHeight, nil
}
