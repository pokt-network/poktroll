package query

import (
	"context"

	"cosmossdk.io/depinject"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.BankQueryClient = (*bankQuerier)(nil)

// bankQuerier is a wrapper around the banktypes.QueryClient that enables the
// querying of onchain balance information.
type bankQuerier struct {
	clientConn  grpc.ClientConn
	bankQuerier banktypes.QueryClient

	// balancesCache caches bankQueryClient.GetBalance requests
	balancesCache KeyValueCache[*sdk.Coin]
}

// NewBankQuerier returns a new instance of a client.BankQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
func NewBankQuerier(deps depinject.Config) (client.BankQueryClient, error) {
	bq := &bankQuerier{}

	if err := depinject.Inject(
		deps,
		&bq.clientConn,
		&bq.balancesCache,
	); err != nil {
		return nil, err
	}

	bq.bankQuerier = banktypes.NewQueryClient(bq.clientConn)

	return bq, nil
}

// GetBalance returns the uPOKT balance of a given address
func (bq *bankQuerier) GetBalance(
	ctx context.Context,
	address string,
) (*sdk.Coin, error) {
	// Check if the account balance is present in the cache.
	if balance, found := bq.balancesCache.Get(address); found {
		return balance, nil
	}

	// Query the blockchain for the balance record
	req := &banktypes.QueryBalanceRequest{Address: address, Denom: volatile.DenomuPOKT}
	res, err := bq.bankQuerier.Balance(ctx, req)
	if err != nil {
		return nil, ErrQueryBalanceNotFound.Wrapf("address: %s [%s]", address, err)
	}

	// Cache the balance for future queries
	bq.balancesCache.Set(address, res.Balance)
	return res.Balance, nil
}
