package proxy

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ring "github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/x/service/types"
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

	// TODO_IN_THIS_PR: Either slow down blocks, or increase numBlocksPerSession,
	// or create a ticket related to session rollovers and link to it here.
	// currentBlock := rp.blockClient.LastNBlocks(ctx, 1)[0]
	// session, err := rp.sessionQuerier.GetSession(ctx, appAddress, service.Id, currentBlock.Height())
	// session, err := rp.sessionQuerier.GetSession(ctx, appAddress, service.Id, relayRequest.Meta.SessionHeader.SessionStartBlockHeight)
	currentBlock := rp.blockClient.LastNBlocks(ctx, 1)[0]
	session, err := rp.sessionQuerier.GetSession(ctx, appAddress, service.Id, currentBlock.Height())

	if err != nil {
		return err
	}

	// Since the retrieved sessionId was in terms of:
	// - the current block height (which is not provided by the relayRequest)
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
