package proxy

import (
	"context"
	"log"

	sdkerrors "cosmossdk.io/errors"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	"github.com/cometbft/cometbft/crypto"
	"github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// VerifyRelayRequest is a shared method used by RelayServers to check the relay request signature and session validity.
func (rp *relayerProxy) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *types.RelayRequest,
	service *sharedtypes.Service,
) error {
	// extract the relay request's ring signature
	log.Printf("DEBUG: Verifying relay request signature...")
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
	signableBz, err := relayRequest.GetSignableBytes()
	if err != nil {
		return sdkerrors.Wrapf(ErrRelayerProxyInvalidRelayRequest, "error getting signable bytes: %v", err)
	}

	hash := crypto.Sha256(signableBz)
	var hash32 [32]byte
	copy(hash32[:], hash)

	// verify the relay request's signature
	if valid := ringSig.Verify(hash32); !valid {
		return sdkerrors.Wrapf(
			ErrRelayerProxyInvalidRelayRequestSignature,
			"invalid ring signature",
		)
	}

	// Query for the current session to check if relayRequest sessionId matches the current session.
	log.Printf("DEBUG: Verifying relay request session...")
	currentBlock := rp.blockClient.LatestBlock(ctx)
	sessionQuery := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		Service:            service,
		BlockHeight:        currentBlock.Height(),
	}
	sessionResponse, err := rp.sessionQuerier.GetSession(ctx, sessionQuery)
	if err != nil {
		return err
	}

	session := sessionResponse.Session

	// Since the retrieved sessionId was in terms of:
	// - the current block height (which is not provided by the relayRequest)
	// - serviceId (which is not provided by the relayRequest)
	// - applicationAddress (which is used to to verify the relayRequest signature)
	// we can reduce the session validity check to checking if the retrieved session's sessionId
	// matches the relayRequest sessionId.
	// TODO_INVESTIGATE: Revisit the assumptions above at some point in the future, but good enough for now.
	if session.SessionId != relayRequest.Meta.SessionHeader.SessionId {
		return ErrRelayerProxyInvalidSession.Wrapf("%+v", session)
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	for _, supplier := range session.Suppliers {
		if supplier.Address == rp.supplierAddress {
			return nil
		}
	}

	return ErrRelayerProxyInvalidSupplier
}
