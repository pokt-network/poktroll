package appclient

import (
	"context"

	"github.com/cometbft/cometbft/crypto"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"pocket/x/service/types"
)

// verifyResponse verifies the relay response signature.
func (app *appClient) verifyResponse(
	ctx context.Context,
	supplierAddress string,
	relayResponse *types.RelayResponse,
) error {
	// Query for the supplier account to get the application's public key to verify the relay request signature.
	accQueryReq := &accounttypes.QueryAccountRequest{Address: supplierAddress}
	accQueryRes, err := app.accountQuerier.Account(ctx, accQueryReq)
	if err != nil {
		return err
	}

	// Marshal the relay response payload to bytes and get the hash.
	var payloadBz []byte
	_, err = relayResponse.Payload.MarshalTo(payloadBz)
	if err != nil {
		return err
	}
	hash := crypto.Sha256(payloadBz)

	// accQueryRes.Account.Value is a protobuf Any type that should be unmarshaled into an AccountI interface.
	// TODO_TECHDEBT: Make sure our `AccountI`/`any` unmarshalling is correct.
	// See https://github.com/pokt-network/poktroll/pull/101/files/edbd628e9146e232ef58c71cfa8f4be2135cdb50..fbba10626df79f6bf6e2218513dfdeb40a629790#r1372464439
	var account accounttypes.AccountI
	if err := app.clientCtx.Codec.UnmarshalJSON(accQueryRes.Account.Value, account); err != nil {
		return err
	}

	// Verify the relay response signature.
	if !account.GetPubKey().VerifySignature(hash, relayResponse.Meta.SupplierSignature) {
		return ErrInvalidRelayResponseSignature
	}

	return nil
}
