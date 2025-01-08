package types

import (
	"context"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.AccountQueryClient = (*AccountKeeperQueryClient)(nil)

// AccountKeeperQueryClient is a thin wrapper around the AccountKeeper.
// It does not rely on the QueryClient, and therefore does not make any
// network requests as in the off-chain implementation.
type AccountKeeperQueryClient struct {
	keeper             AccountKeeper
	accountPubKeyCache map[string]cryptotypes.PubKey
}

// NewAccountKeeperQueryClient returns a new AccountQueryClient that is backed
// by an AccountKeeper instance.
// It is used by the PubKeyClient to get the public key that corresponds to the
// provided address.
// It should be injected into the PubKeyClient when initialized from within the a keeper.
func NewAccountKeeperQueryClient(accountKeeper AccountKeeper) client.AccountQueryClient {
	return &AccountKeeperQueryClient{
		keeper:             accountKeeper,
		accountPubKeyCache: make(map[string]cryptotypes.PubKey),
	}
}

// GetAccount returns the account associated with the provided address.
func (accountQueryClient *AccountKeeperQueryClient) GetAccount(
	ctx context.Context,
	addr string,
) (account types.AccountI, err error) {
	addrBz, err := types.AccAddressFromBech32(addr)
	if err != nil {
		return nil, err
	}

	// keeper.GetAccount panics if the account is not found.
	// Capture the panic and return an error if one occurs.
	defer func() {
		if r := recover(); r != nil {
			err = ErrProofPubKeyNotFound
			account = nil
		}
	}()

	// Retrieve an account from the account keeper.
	account = accountQueryClient.keeper.GetAccount(ctx, addrBz)

	return account, err
}

// GetPubKeyFromAddress returns the public key of the given address.
// It uses the accountQuerier to get the account and then returns its public key.
func (accountQueryClient *AccountKeeperQueryClient) GetPubKeyFromAddress(
	ctx context.Context,
	address string,
) (cryptotypes.PubKey, error) {
	if acc, found := accountQueryClient.accountPubKeyCache[address]; found {
		return acc, nil
	}

	acc, err := accountQueryClient.GetAccount(ctx, address)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, ErrProofAccNotFound.Wrapf("account not found for address %s", address)
	}

	// If the account's public key is nil, then return an error.
	pubKey := acc.GetPubKey()
	if pubKey == nil {
		return nil, ErrProofPubKeyNotFound
	}

	accountQueryClient.accountPubKeyCache[address] = pubKey

	return pubKey, nil
}

func (accountQueryClient *AccountKeeperQueryClient) ResetCache() {
	clear(accountQueryClient.accountPubKeyCache)
}
