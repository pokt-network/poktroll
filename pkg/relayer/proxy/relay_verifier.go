package proxy

import (
	"github.com/cometbft/cometbft/crypto"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"golang.org/x/exp/slices"

	"context"
	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

// VerifyRelayRequest is a shared method used by RelayServers to check the relay request signature and session validity.
func (rp *relayerProxy) VerifyRelayRequest(
	ctx context.Context,
	relayRequest *types.RelayRequest,
	serviceId *sharedtypes.ServiceId,
) error {
	// Query for the application account to get the application's public key to verify the relay request signature.
	applicationAddress := relayRequest.Meta.SessionHeader.ApplicationAddress
	accQueryReq := &accounttypes.QueryAccountRequest{Address: applicationAddress}
	accQueryRes, err := rp.accountsQuerier.Account(ctx, accQueryReq)
	if err != nil {
		return err
	}

	var payloadBz []byte
	_, err = relayRequest.Payload.MarshalTo(payloadBz)
	if err != nil {
		return err
	}
	hash := crypto.Sha256(payloadBz)

	// accountResponse.Account.Value is a protobuf Any type that should be unmarshaled into an AccountI interface.
	// TODO_TECHDEBT: Make sure our `AccountI`/`any` unmarshalling is correct. See https://github.com/pokt-network/poktroll/pull/101/files/edbd628e9146e232ef58c71cfa8f4be2135cdb50..fbba10626df79f6bf6e2218513dfdeb40a629790#r1372464439
	var account accounttypes.AccountI
	if err := rp.clientCtx.Codec.UnmarshalJSON(accQueryRes.Account.Value, account); err != nil {
		return err
	}

	if !account.GetPubKey().VerifySignature(hash, relayRequest.Meta.Signature) {
		return ErrInvalidRelayRequestSignature
	}

	// Query for the current session to check if relayRequest sessionId matches the current session.
	currentBlock := rp.blockClient.LatestBlock(ctx)
	sessionQuery := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: applicationAddress,
		ServiceId:          &sessiontypes.ServiceId{Id: serviceId},
		BlockHeight:        currentBlock.Height(),
	}
	sessionResponse, err := rp.sessionQuerier.GetSession(ctx, sessionQuery)
	session := sessionResponse.Session

	// Since the retrieved sessionId was in terms of:
	// - the current block height (which is not provided by the relayRequest)
	// - serviceId (which is not provided by the relayRequest)
	// - applicationAddress (which is used to to verify the relayRequest signature)
	// we can reduce the session validity check to checking if the retrieved session's sessionId
	// matches the relayRequest sessionId.
	// TODO_INVESTIGATE: Revisit the assumptions above at some point in the future, but good enough for now.
	if session.SessionId != relayRequest.Meta.SessionHeader.SessionId {
		return ErrInvalidSession
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	if !slices.Contains(session.Suppliers, rp.supplierAddress) {
		return ErrInvalidSupplier
	}

	return nil
}
