package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
)

var _ client.BankQueryClient = (*bankQuerier)(nil)

// bankQuerier is a wrapper around the banktypes.QueryClient that enables the
// querying of onchain balance information.
type bankQuerier struct {
	clientConn  grpc.ClientConn
	bankQuerier banktypes.QueryClient
	logger      polylog.Logger

	// balancesCache caches bankQueryClient.GetBalance requests
	balancesCache cache.KeyValueCache[Balance]
	// Mutex to protect balancesCache access
	balancesMutex sync.Mutex
}

// NewBankQuerier returns a new instance of a client.BankQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
// - polylog.Logger
// - cache.KeyValueCache[Balance]
func NewBankQuerier(deps depinject.Config) (client.BankQueryClient, error) {
	bq := &bankQuerier{}

	if err := depinject.Inject(
		deps,
		&bq.clientConn,
		&bq.logger,
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
	logger := bq.logger.With("query_client", "bank", "method", "GetBalance")

	// Check if the account balance is present in the cache.
	if balance, found := bq.balancesCache.Get(address); found {
		logger.Debug().Msgf("cache HIT for account balance with address: %s", address)
		return balance, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	bq.balancesMutex.Lock()
	defer bq.balancesMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if balance, found := bq.balancesCache.Get(address); found {
		logger.Debug().Msgf("cache HIT for account balance with address after lock: %s", address)
		return balance, nil
	}

	logger.Debug().Msgf("cache MISS for account balance with address: %s", address)

	// Query the blockchain for the balance record
	req := &banktypes.QueryBalanceRequest{Address: address, Denom: pocket.DenomuPOKT}
	res, err := retry.Call(ctx, func() (*banktypes.QueryBalanceResponse, error) {
		return bq.bankQuerier.Balance(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, ErrQueryBalanceNotFound.Wrapf("address: %s [%s]", address, err)
	}

	// Cache the balance for future queries
	bq.balancesCache.Set(address, res.Balance)
	return res.Balance, nil
}
