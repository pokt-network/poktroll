package appgateserver

import (
	"context"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/x/service/types"
)

// verifyResponse verifies the relay response signature.
func (app *appGateServer) verifyResponse(
	ctx context.Context,
	supplierAddress string,
	relayResponse *types.RelayResponse,
) error {
	// Get the supplier's public key.
	supplierPubKey, err := app.getSupplierPubKeyFromAddress(ctx, supplierAddress)
	if err != nil {
		return err
	}

	// Extract the supplier's signature
	if relayResponse.Meta == nil {
		payload := relayResponse.GetPayload()
		payloadBz := make([]byte, payload.Size())
		if _, err := payload.MarshalTo(payloadBz); err != nil {
			return ErrAppGateEmptyRelayResponseMeta.Wrapf(
				"unable to marshal relay response payload: %s", err,
			)
		}
		return ErrAppGateEmptyRelayResponseSignature.Wrapf(
			"response payload: %s", relayResponse.Payload,
		)
	}
	supplierSignature := relayResponse.Meta.SupplierSignature

	// Get the relay response signable bytes and hash them.
	responseBz, err := relayResponse.GetSignableBytes()
	if err != nil {
		return err
	}
	hash := crypto.Sha256(responseBz)

	// Verify the relay response signature.
	if !supplierPubKey.VerifySignature(hash, supplierSignature) {
		return ErrAppGateInvalidRelayResponseSignature
	}

	return nil
}

// getSupplierPubKeyFromAddress gets the supplier's public key from the cache or
// queries if it is not found. The public key is then cached before being returned.
func (app *appGateServer) getSupplierPubKeyFromAddress(
	ctx context.Context,
	supplierAddress string,
) (cryptotypes.PubKey, error) {
	supplierPubKey, ok := app.supplierAccountCache[supplierAddress]
	if ok {
		return supplierPubKey, nil
	}

	// Query for the supplier account to get the application's public key
	// to verify the relay request signature.
	accQueryReq := &accounttypes.QueryAccountRequest{Address: supplierAddress}
	accQueryRes, err := app.accountQuerier.Account(ctx, accQueryReq)
	if err != nil {
		return nil, err
	}

	// Unmarshal the query response into a BaseAccount.
	var acc accounttypes.AccountI
	reg := codectypes.NewInterfaceRegistry()
	accounttypes.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	if err := cdc.UnpackAny(accQueryRes.Account, &acc); err != nil {
		return nil, err
	}

	fetchedPubKey := acc.GetPubKey()
	// Cache the retrieved public key.
	app.supplierAccountCache[supplierAddress] = fetchedPubKey

	return fetchedPubKey, nil
}
