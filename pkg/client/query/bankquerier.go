package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

var _ client.BankQueryClient = (*bankQuerier)(nil)

// bankQuerier is a wrapper around the banktypes.QueryClient that enables the
// querying of onchain balance information.
type bankQuerier struct {
	clientConn  grpc.ClientConn
	bankQuerier banktypes.QueryClient

	blockClient client.BlockClient
	bankCache   map[string]*sdk.Coin
	bankCacheMu sync.Mutex
}

// NewBankQuerier returns a new instance of a client.BankQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
func NewBankQuerier(ctx context.Context, deps depinject.Config) (client.BankQueryClient, error) {
	bq := &bankQuerier{}

	if err := depinject.Inject(
		deps,
		&bq.blockClient,
		&bq.clientConn,
	); err != nil {
		return nil, err
	}

	bq.bankQuerier = banktypes.NewQueryClient(bq.clientConn)

	bq.bankCache = make(map[string]*sdk.Coin)

	channel.ForEach(
		ctx,
		bq.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			bq.bankCacheMu.Lock()
			defer bq.bankCacheMu.Unlock()

			bq.bankCache = make(map[string]*sdk.Coin)
		},
	)

	return bq, nil
}

// GetBalance returns the uPOKT balance of a given address
func (bq *bankQuerier) GetBalance(
	ctx context.Context,
	address string,
) (*sdk.Coin, error) {
	bq.bankCacheMu.Lock()
	defer bq.bankCacheMu.Unlock()

	if foundBalance, isBalanceFound := bq.bankCache[address]; isBalanceFound {
		return foundBalance, nil
	}
	// Query the blockchain for the balance record
	req := &banktypes.QueryBalanceRequest{Address: address, Denom: volatile.DenomuPOKT}
	res, err := bq.bankQuerier.Balance(ctx, req)
	if err != nil {
		return nil, ErrQueryBalanceNotFound.Wrapf("address: %s [%s]", address, err)
	}

	bq.bankCache[address] = res.Balance

	return res.Balance, nil
}
