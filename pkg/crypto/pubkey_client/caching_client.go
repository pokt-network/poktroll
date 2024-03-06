package pubkeyclient

import (
	"context"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
)

var _ crypto.PubKeyClient = (*pubKeyCachingClient)(nil)

// pubKeyCachingClient is an implementation of the PubKeyClient that uses an account
// querier to fill the cache of public keys for future use.
type pubKeyCachingClient struct {
	// accountQuerier is the querier for the account module, and is used to get
	// the public key of an address.
	accountQuerier client.AccountQueryClient

	// pubKeyCache is a cache of public keys for addresses.
	// It is used to avoid querying the account module for the same public key
	pubKeyCache map[string]cryptotypes.PubKey
}

// NewPubKeyClientWithCache creates a new PubKeyClient with the given dependencies.
// It is intended to be used in the off-chain environment, where accounts have
// to be fetched over the network.
//
// Required dependencies:
// - client.AccountQueryClient
func NewPubKeyClientWithCache(deps depinject.Config) (crypto.PubKeyClient, error) {
	pcc := new(pubKeyCachingClient)

	if err := depinject.Inject(
		deps,
		&pcc.accountQuerier,
	); err != nil {
		return nil, err
	}

	return pcc, nil
}

// GetPubKeyFromAddress returns the public key of the given address.
// It uses the accountQuerier to get the account if it is not already in the cache.
// If the account does not have a public key, it returns an error.
func (pcc *pubKeyCachingClient) GetPubKeyFromAddress(
	ctx context.Context,
	address string,
) (cryptotypes.PubKey, error) {
	if pubKey, ok := pcc.pubKeyCache[address]; ok {
		return pubKey, nil
	}

	acc, err := pcc.accountQuerier.GetAccount(ctx, address)
	if err != nil {
		return nil, err
	}

	pubKey := acc.GetPubKey()
	if pubKey == nil {
		return nil, ErrPubKeyClientEmptyPubKey
	}

	pcc.pubKeyCache[address] = pubKey
	return pubKey, nil
}

// VerifySignature verifies a signature against the signable bytes and the public
// key of the given address.
// It uses GetPubKeyFromAddress to get potentially cached public keys and then
// verifies the signature.
func (pcc *pubKeyCachingClient) VerifySignature(
	ctx context.Context,
	address string,
	signature []byte,
	signedBz []byte,
) error {
	// Get the address' public key.
	supplierPubKey, err := pcc.GetPubKeyFromAddress(ctx, address)
	if err != nil {
		return err
	}

	// Verify whether the signature is valid for the given public key and signed bytes
	if !supplierPubKey.VerifySignature(signedBz[:], signature) {
		return ErrPubKeyClientInvalidSignature
	}

	return nil
}
