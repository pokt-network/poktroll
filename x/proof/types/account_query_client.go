package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.AccountQueryClient = (*AccountKeeperQueryClient)(nil)

type AccountKeeperQueryClient struct {
	keeper AccountKeeper
}

// NewAccountKeeperQueryClient returns a new AccountQueryClient that is backed
// by an AccountKeeper instance.
// It is used by the PubKeyClient to get the public key that corresponds to the
// provided address.
// This implementation is a thin wrapper around the AccountKeeper and does
// not rely on the QueryClient contrariwise to the off-chain implementation.
// It should be injected into the PubKeyClient when initialized from within the a keeper.
func NewAccountKeeperQueryClient(accountKeeper AccountKeeper) client.AccountQueryClient {
	return &AccountKeeperQueryClient{keeper: accountKeeper}
}

func (accountQueryClient *AccountKeeperQueryClient) GetAccount(
	ctx context.Context,
	addr string,
) (accounttypes.AccountI, error) {
	addrBz, err := types.AccAddressFromBech32(addr)
	if err != nil {
		return nil, err
	}

	return accountQueryClient.keeper.GetAccount(ctx, addrBz), nil
}
