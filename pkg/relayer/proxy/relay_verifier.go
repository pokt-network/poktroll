package proxy

import (
	"context"

	"github.com/cometbft/cometbft/crypto"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

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
	// Query for the application account to get the application's public key to verify the relay request signature.
	applicationAddress := relayRequest.Meta.SessionHeader.ApplicationAddress
	accQueryReq := &accounttypes.QueryAccountRequest{Address: applicationAddress}
	accQueryRes, err := rp.accountsQuerier.Account(ctx, accQueryReq)
	if err != nil {
		return err
	}

	var payloadBz []byte
	if _, err = relayRequest.Payload.MarshalTo(payloadBz); err != nil {
		return err
	}
	hash := crypto.Sha256(payloadBz)

	account := new(accounttypes.BaseAccount)
	if err := account.Unmarshal(accQueryRes.Account.Value); err != nil {
		return err
	}

	if !account.GetPubKey().VerifySignature(hash, relayRequest.Meta.Signature) {
		return ErrRelayerProxyInvalidRelayRequestSignature
	}

	// Query for the current session to check if relayRequest sessionId matches the current session.
	currentBlock := rp.blockClient.LatestBlock(ctx)
	sessionQuery := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: applicationAddress,
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
		return ErrRelayerProxyInvalidSession
	}

	// Check if the relayRequest is allowed to be served by the relayer proxy.
	for _, supplier := range session.Suppliers {
		if supplier.Address == rp.supplierAddress {
			return nil
		}
	}

	return ErrRelayerProxyInvalidSupplier
}
