package appserver

import (
	"context"

	"github.com/cometbft/cometbft/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/x/service/types"
)

// verifyResponse verifies the relay response signature.
func (app *appServer) verifyResponse(
	ctx context.Context,
	supplierAddress string,
	relayResponse *types.RelayResponse,
) error {
	pubKey, err := app.getSupplierPubKeyFromAddress(ctx, supplierAddress)
	if err != nil {
		return err
	}

	signature := relayResponse.Meta.SupplierSignature
	relayResponse.Meta.SupplierSignature = nil

	// Marshal the relay response payload to bytes and get the hash.
	var payloadBz []byte
	if _, err = relayResponse.Payload.MarshalTo(payloadBz); err != nil {
		return err
	}
	hash := crypto.Sha256(payloadBz)

	// Verify the relay response signature.
	if !pubKey.VerifySignature(hash, signature) {
		return ErrInvalidRelayResponseSignature
	}

	return nil
}

// getSupplierPubKeyFromAddress gets the supplier's public key from the cache or queries
// if it is not found.
// The public key is then cached before being returned.
func (app *appServer) getSupplierPubKeyFromAddress(
	ctx context.Context,
	supplierAddress string,
) (cryptotypes.PubKey, error) {
	pubKey, ok := app.supplierAccountCache[supplierAddress]
	if ok {
		return pubKey, nil
	}

	// Query for the supplier account to get the application's public key to verify the relay request signature.
	accQueryReq := &accounttypes.QueryAccountRequest{Address: supplierAddress}
	accQueryRes, err := app.accountQuerier.Account(ctx, accQueryReq)
	if err != nil {
		return nil, err
	}

	// Unmarshal the query response into a BaseAccount.
	account := new(accounttypes.BaseAccount)
	if err := account.Unmarshal(accQueryRes.Account.Value); err != nil {
		return nil, err
	}

	fetchedPubKey := account.GetPubKey()
	// Cache the retrieved public key.
	app.supplierAccountCache[supplierAddress] = fetchedPubKey

	return fetchedPubKey, nil
}
