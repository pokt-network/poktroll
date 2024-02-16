package sdk

import (
	"context"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/x/service/types"
)

// verifyResponse verifies the relay response signature.
func (sdk *poktrollSDK) verifyResponse(
	ctx context.Context,
	supplierAddress string,
	relayResponse *types.RelayResponse,
) error {
	logger := polylog.Ctx(context.Background())

	// Get the supplier's public key.
	supplierPubKey, err := sdk.getSupplierPubKeyFromAddress(ctx, supplierAddress)
	if err != nil {
		return err
	}

	// Extract the supplier's signature
	if relayResponse.Meta == nil {
		return ErrSDKEmptyRelayResponseSignature.Wrapf(
			"response payload: %s", relayResponse.Payload,
		)
	}
	supplierSignature := relayResponse.Meta.SupplierSignature

	// Get the relay response signable bytes and hash them.
	responseSignableBz, err := relayResponse.GetSignableBytesHash()
	if err != nil {
		return err
	}

	logger.Debug().
		Str("supplier", supplierAddress).
		Str("application", relayResponse.Meta.SessionHeader.ApplicationAddress).
		Str("service", relayResponse.Meta.SessionHeader.Service.Id).
		Int64("end_height", relayResponse.Meta.SessionHeader.SessionEndBlockHeight).
		Msg("About to verify relay response signature.")

	// Verify the relay response signature.
	if !supplierPubKey.VerifySignature(responseSignableBz[:], supplierSignature) {
		return ErrSDKInvalidRelayResponseSignature
	}

	return nil
}

// getSupplierPubKeyFromAddress gets the supplier's public key from the cache or
// queries if it is not found. The public key is then cached before being returned.
func (sdk *poktrollSDK) getSupplierPubKeyFromAddress(
	ctx context.Context,
	supplierAddress string,
) (cryptotypes.PubKey, error) {
	supplierPubKey, ok := sdk.supplierAccountCache[supplierAddress]
	if ok {
		return supplierPubKey, nil
	}

	// Query for the supplier account to get the application's public key
	// to verify the relay request signature.
	acc, err := sdk.accountQuerier.GetAccount(ctx, supplierAddress)
	if err != nil {
		return nil, err
	}

	fetchedPubKey := acc.GetPubKey()
	if fetchedPubKey == nil {
		return nil, ErrSDKEmptySupplierPubKey
	}

	// Cache the retrieved public key.
	sdk.supplierAccountCache[supplierAddress] = fetchedPubKey

	return fetchedPubKey, nil
}
