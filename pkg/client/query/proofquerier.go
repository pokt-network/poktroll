package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying of onchain proof module params.
type proofQuerier struct {
	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient

	blockClient      client.BlockClient
	proofParamsCache client.ProofParams
	bankCacheMu      sync.Mutex
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewProofQuerier(ctx context.Context, deps depinject.Config) (client.ProofQueryClient, error) {
	querier := &proofQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.blockClient,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	channel.ForEach(
		ctx,
		querier.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			querier.bankCacheMu.Lock()
			defer querier.bankCacheMu.Unlock()

			querier.proofParamsCache = nil
		},
	)

	return querier, nil
}

// GetParams queries the chain for the current proof module parameters.
func (pq *proofQuerier) GetParams(
	ctx context.Context,
) (client.ProofParams, error) {
	pq.bankCacheMu.Lock()
	defer pq.bankCacheMu.Unlock()

	if pq.proofParamsCache != nil {
		return pq.proofParamsCache, nil
	}
	req := &prooftypes.QueryParamsRequest{}
	res, err := pq.proofQuerier.Params(ctx, req)
	if err != nil {
		return nil, err
	}

	pq.proofParamsCache = &res.Params
	return pq.proofParamsCache, nil
}
