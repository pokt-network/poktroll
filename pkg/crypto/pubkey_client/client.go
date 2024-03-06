package pubkeyclient

import (
	"context"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
)

var _ crypto.PubKeyClient = (*pubKeyClient)(nil)

// pubKeyClient is an implementation of the PubKeyClient that uses an account
// querier to get the public key of an address.
type pubKeyClient struct {
	// accountQuerier is the querier for the account module, and is used to get
	// the public key of an address.
	accountQuerier client.AccountQueryClient
}

// NewPubKeyClient creates a new PubKeyClient with the given dependencies.
// The querier is injected using depinject and may be specific to the
// environment in which the pubKeyClient is initialized as on-chain and off-chain
// environments may have different queriers.
//
// Required dependencies:
// - client.AccountQueryClient
func NewPubKeyClient(deps depinject.Config) (crypto.PubKeyClient, error) {
	pc := new(pubKeyClient)

	if err := depinject.Inject(
		deps,
		&pc.accountQuerier,
	); err != nil {
		return nil, err
	}

	return pc, nil
}

// GetPubKeyFromAddress returns the public key of the given address.
// It uses the accountQuerier to get the account and then returns its public key.
// If the account does not have a public key, it returns an error.
func (pc *pubKeyClient) GetPubKeyFromAddress(ctx context.Context, address string) (cryptotypes.PubKey, error) {
	acc, err := pc.accountQuerier.GetAccount(ctx, address)
	if err != nil {
		return nil, err
	}

	pubKey := acc.GetPubKey()
	if pubKey == nil {
		return nil, ErrPubKeyClientEmptyPubKey
	}

	return pubKey, nil
}

// VerifySignature verifies a signature against the signable bytes and the public
// key of the given address.
// It uses GetPubKeyFromAddress to get the public key and then verifies the signature.
func (pc *pubKeyClient) VerifySignature(
	ctx context.Context,
	address string,
	signature []byte,
	signedBz []byte,
) error {
	// Get the address' public key.
	supplierPubKey, err := pc.GetPubKeyFromAddress(ctx, address)
	if err != nil {
		return err
	}

	// Verify whether the signature is valid for the given public key and signed bytes
	if !supplierPubKey.VerifySignature(signedBz[:], signature) {
		return ErrPubKeyClientInvalidSignature
	}

	return nil
}
