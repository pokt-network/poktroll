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
