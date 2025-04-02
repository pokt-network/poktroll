package query

import (
	"context"
	"strconv"

	"cosmossdk.io/depinject"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SharedQueryClient = (*sharedQuerier)(nil)

// sharedQuerier is a wrapper around the sharedtypes.QueryClient that enables the
// querying of onchain shared information through a single exposed method
// which returns an sharedtypes.Session struct
type sharedQuerier struct {
	clientConn    grpc.ClientConn
	sharedQuerier sharedtypes.QueryClient
	blockQuerier  client.BlockQueryClient
	logger        polylog.Logger

	// blockHashCache caches blockQuerier.Block requests
	blockHashCache cache.KeyValueCache[BlockHash]
	// paramsCache caches sharedQueryClient.Params requests
	paramsCache client.ParamsCache[sharedtypes.Params]
}

// NewSharedQuerier returns a new instance of a client.SharedQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - client.BlockQueryClient
func NewSharedQuerier(deps depinject.Config) (client.SharedQueryClient, error) {
	querier := &sharedQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.logger,
		&querier.blockQuerier,
		&querier.blockHashCache,
		&querier.paramsCache,
	); err != nil {
		return nil, err
	}

	querier.sharedQuerier = sharedtypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetParams queries & returns the shared module onchain parameters.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
func (sq *sharedQuerier) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetParams")

	// Get the params from the cache if they exist.
	if params, found := sq.paramsCache.Get(); found {
		logger.Debug().Msg("cache hit for shared params")
		return &params, nil
	}

	logger.Debug().Msg("cache miss for shared params")

	req := &sharedtypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*sharedtypes.QueryParamsResponse, error) {
		return sq.sharedQuerier.Params(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	// Update the cache with the newly retrieved params.
	sq.paramsCache.Set(res.Params)
	return &res.Params, nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
// we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
//
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
// we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetSessionGracePeriodEndHeight(sharedParams, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetEarliestSupplierClaimCommitHeight")

	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight)

	// Check if the block hash is already in the cache.
	blockHashCacheKey := getBlockHashCacheKey(claimWindowOpenHeight)
	claimWindowOpenBlockHash, found := sq.blockHashCache.Get(blockHashCacheKey)
	if !found {
		logger.Debug().Msgf("cache miss for blockHeight: %s", blockHashCacheKey)

		claimWindowOpenBlock, err := retry.Call(ctx, func() (*cometrpctypes.ResultBlock, error) {
			return sq.blockQuerier.Block(ctx, &claimWindowOpenHeight)
		}, retry.GetStrategy(ctx))
		if err != nil {
			return 0, err
		}

		// Cache the block hash for future use.
		// NB: Byte slice representation of block hashes don't need to be normalized.
		claimWindowOpenBlockHash = claimWindowOpenBlock.BlockID.Hash.Bytes()
		sq.blockHashCache.Set(blockHashCacheKey, claimWindowOpenBlockHash)
	} else {
		logger.Debug().Msgf("cache hit for blockHeight: %s", blockHashCacheKey)
	}

	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		queryHeight,
		claimWindowOpenBlockHash,
		supplierOperatorAddr,
	), nil
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetEarliestSupplierProofCommitHeight")

	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight)

	blockHashCacheKey := getBlockHashCacheKey(proofWindowOpenHeight)
	proofWindowOpenBlockHash, found := sq.blockHashCache.Get(blockHashCacheKey)

	if !found {
		logger.Debug().Msgf("cache miss for blockHeight: %s", blockHashCacheKey)

		proofWindowOpenBlock, err := retry.Call(ctx, func() (*cometrpctypes.ResultBlock, error) {
			return sq.blockQuerier.Block(ctx, &proofWindowOpenHeight)
		}, retry.GetStrategy(ctx))
		if err != nil {
			return 0, err
		}

		// Cache the block hash for future use.
		proofWindowOpenBlockHash = proofWindowOpenBlock.BlockID.Hash.Bytes()
		sq.blockHashCache.Set(blockHashCacheKey, proofWindowOpenBlockHash)
	} else {
		logger.Debug().Msgf("cache hit for blockHeight: %s", blockHashCacheKey)
	}

	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		queryHeight,
		proofWindowOpenBlockHash,
		supplierOperatorAddr,
	), nil
}

// GetComputeUnitsToTokensMultiplier returns the multiplier used to convert compute units to tokens.
//
// TODO_MAINNET_MIGRATION(@bryanchriswhite, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetComputeUnitsToTokensMultiplier(ctx context.Context) (uint64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return sharedParams.GetComputeUnitsToTokensMultiplier(), nil
}

// getBlockHashCacheKey constructs the cache key for a block hash by string formatting the block height.
func getBlockHashCacheKey(height int64) string {
	return strconv.FormatInt(height, 10)
}
