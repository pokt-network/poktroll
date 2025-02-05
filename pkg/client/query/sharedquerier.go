package query

import (
	"context"
	"strconv"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
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

	// blockHashCache caches blockQuerier.Block requests
	blockHashCache KeyValueCache[[]byte]
	// paramsCache caches sharedQueryClient.Params requests
	paramsCache ParamsCache[sharedtypes.Params]
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
		&querier.blockQuerier,
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
	// Get the params from the cache if they exist.
	if params, found := sq.paramsCache.Get(); found {
		return &params, nil
	}

	req := &sharedtypes.QueryParamsRequest{}
	res, err := sq.sharedQuerier.Params(ctx, req)
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
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
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
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
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
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
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
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight)

	// Check if the block hash is already in the cache.
	blockHashCacheKey := getBlockHashKacheKey(claimWindowOpenHeight)
	claimWindowOpenBlockHash, found := sq.blockHashCache.Get(blockHashCacheKey)
	if !found {
		claimWindowOpenBlock, err := sq.blockQuerier.Block(ctx, &claimWindowOpenHeight)
		if err != nil {
			return 0, err
		}

		// Cache the block hash for future use.
		// NB: Byte slice representation of block hashes don't need to be normalized.
		claimWindowOpenBlockHash = claimWindowOpenBlock.BlockID.Hash.Bytes()
		sq.blockHashCache.Set(blockHashCacheKey, claimWindowOpenBlockHash)
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
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	blockHashCacheKey := getBlockHashKacheKey(queryHeight)
	proofWindowOpenBlockHash, found := sq.blockHashCache.Get(blockHashCacheKey)

	if !found {
		// Fetch the block at the proof window open height. Its hash is used as part
		// of the seed to the pseudo-random number generator.
		proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight)
		proofWindowOpenBlock, err := sq.blockQuerier.Block(ctx, &proofWindowOpenHeight)
		if err != nil {
			return 0, err
		}

		// Cache the block hash for future use.
		proofWindowOpenBlockHash = proofWindowOpenBlock.BlockID.Hash.Bytes()
		sq.blockHashCache.Set(blockHashCacheKey, proofWindowOpenBlockHash)
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
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
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

// getBlockHashKacheKey constructs the cache key for a block hash.
func getBlockHashKacheKey(height int64) string {
	return strconv.FormatInt(height, 10)
}
