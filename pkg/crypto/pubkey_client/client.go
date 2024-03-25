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
	// accountQuerier is the querier for the account module, it is used to get
	// the public key corresponding to an address.
	accountQuerier client.AccountQueryClient
}

// NewPubKeyClient creates a new PubKeyClient with the given dependencies.
// The querier is injected using depinject and has to be specific to the
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
// It retrieves the corresponding account by querying for it and returns
// the associated public key.
func (pc *pubKeyClient) GetPubKeyFromAddress(ctx context.Context, address string) (cryptotypes.PubKey, error) {
	acc, err := pc.accountQuerier.GetAccount(ctx, address)
	if err != nil {
		return nil, err
	}

	// If the account's public key is nil, then return an error.
	pubKey := acc.GetPubKey()
	if pubKey == nil {
		return nil, ErrPubKeyClientEmptyPubKey
	}

	return pubKey, nil
}
