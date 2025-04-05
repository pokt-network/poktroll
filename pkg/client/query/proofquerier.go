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
// - polylog.Logger
// - client.BlockClient
// - client.ParamsCache[prooftypes.Params]
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
	if params, found := pq.paramsCache.GetLatest(); found {
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
	pq.paramsCache.SetAtHeight(res.Params, int64(res.EffectiveBlockHeight))
	return &res.Params, nil
}

// GetParamsAtHeight queries & returns the proof module onchain parameters
// that were in effect at the given block height.
func (sq *proofQuerier) GetParamsAtHeight(ctx context.Context, height int64) (client.ProofParams, error) {
	logger := sq.logger.With("query_client", "proof", "method", "GetParamsAtHeight")

	// Get the params from the cache if they exist.
	if params, found := sq.paramsCache.GetAtHeight(height); found {
		logger.Debug().Msgf("cache hit for proof params at height: %d", height)
		return &params, nil
	}

	logger.Debug().Msgf("cache miss for proof params at height: %d", height)

	req := &prooftypes.QueryParamsAtHeightRequest{Height: uint64(height)}
	res, err := sq.proofQuerier.ParamsAtHeight(ctx, req)
	if err != nil {
		return nil, ErrQueryProofParams.Wrapf("[%v]", err)
	}

	// Update the cache with the newly retrieved params.
	sq.paramsCache.SetAtHeight(res.Params, int64(res.EffectiveBlockHeight))
	return &res.Params, nil
}
