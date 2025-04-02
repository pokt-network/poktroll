package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying of onchain proof module params.
type proofQuerier struct {
	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient
	logger       polylog.Logger

	// paramsCache caches proofQuerier.Params requests
	paramsCache client.ParamsCache[prooftypes.Params]
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewProofQuerier(deps depinject.Config) (client.ProofQueryClient, error) {
	querier := &proofQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.logger,
		&querier.paramsCache,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetParams queries the chain for the current proof module parameters.
func (pq *proofQuerier) GetParams(
	ctx context.Context,
) (client.ProofParams, error) {
	logger := pq.logger.With("query_client", "proof", "method", "GetParams")

	// Get the params from the cache if they exist.
	if params, found := pq.paramsCache.Get(); found {
		logger.Debug().Msg("cache hit for proof params")
		return &params, nil
	}

	logger.Debug().Msg("cache miss proof params")

	req := &prooftypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*prooftypes.QueryParamsResponse, error) {
		return pq.proofQuerier.Params(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved params.
	pq.paramsCache.Set(res.Params)
	return &res.Params, nil
}
