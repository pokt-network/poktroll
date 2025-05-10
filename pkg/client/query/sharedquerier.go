package query

import (
	"context"
	"strconv"
	"sync"

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
	// blockHashMutex to protect cache access patterns for block hashes
	blockHashMutex sync.Mutex

	// paramsCache caches sharedQueryClient.Params requests
	paramsCache client.ParamsCache[sharedtypes.Params]
	// paramsMutex to protect cache access patterns for params
	paramsMutex sync.Mutex
}

// NewSharedQuerier returns a new instance of a client.SharedQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - polylog.Logger
// - client.BlockQueryClient
// - cache.KeyValueCache[BlockHash]
// - client.ParamsCache[sharedtypes.Params]
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
func (sq *sharedQuerier) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetParams")

	// TODO_IN_THIS_PR: Ensure that the latest cached version of the shared module
	// parameters is indeed the latest one, by subscribing to params update events.

	// Get the params from the cache if they exist.
	if params, found := sq.paramsCache.GetLatest(); found {
		logger.Debug().Msg("cache HIT for shared params")
		return &params, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	sq.paramsMutex.Lock()
	defer sq.paramsMutex.Unlock()

	// Double-check the cache after acquiring the lock
	if params, found := sq.paramsCache.GetLatest(); found {
		logger.Debug().Msg("cache HIT for shared params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache MISS for shared params")

	req := &sharedtypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*sharedtypes.QueryParamsResponse, error) {
		return sq.sharedQuerier.Params(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	// Update the cache with the newly retrieved params.
	sq.paramsCache.SetAtHeight(res.Params, int64(res.EffectiveBlockHeight))
	return &res.Params, nil
}

// GetParamsAtHeight queries & returns the shared module onchain parameters
// that were in effect at the given block height.
func (sq *sharedQuerier) GetParamsAtHeight(ctx context.Context, height int64) (*sharedtypes.Params, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetParamsAtHeight")

	// Get the params from the cache if they exist.
	if params, found := sq.paramsCache.GetAtHeight(height); found {
		logger.Debug().Msgf("cache hit for shared params at height: %d", height)
		return &params, nil
	}

	logger.Debug().Msgf("cache miss for shared params at height: %d", height)

	req := &sharedtypes.QueryParamsAtHeightRequest{AtHeight: uint64(height)}
	res, err := sq.sharedQuerier.ParamsAtHeight(ctx, req)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	// Update the cache with the newly retrieved params.
	sq.paramsCache.SetAtHeight(res.Params, int64(res.EffectiveBlockHeight))
	return &res.Params, nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
func (sq *sharedQuerier) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetClaimWindowOpenHeight(sharedParamsUpdates, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
func (sq *sharedQuerier) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetProofWindowOpenHeight(sharedParamsUpdates, queryHeight), nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
func (sq *sharedQuerier) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetSessionGracePeriodEndHeight(sharedParamsUpdates, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
func (sq *sharedQuerier) GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetEarliestSupplierClaimCommitHeight")

	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParamsUpdates, queryHeight)

	// Check if the block hash is already in the cache.
	blockHashCacheKey := getBlockHashCacheKey(claimWindowOpenHeight)
	claimWindowOpenBlockHash, found := sq.blockHashCache.Get(blockHashCacheKey)

	if !found {
		// Use mutex for cache miss pattern
		sq.blockHashMutex.Lock()
		defer sq.blockHashMutex.Unlock()

		// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
		claimWindowOpenBlockHash, found = sq.blockHashCache.Get(blockHashCacheKey)
		if found {
			logger.Debug().Msgf("cache HIT for blockHeight after lock: %s", blockHashCacheKey)
		} else {
			logger.Debug().Msgf("cache MISS for blockHeight: %s", blockHashCacheKey)

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
		}
	} else {
		logger.Debug().Msgf("cache HIT for blockHeight: %s", blockHashCacheKey)
	}

	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParamsUpdates,
		queryHeight,
		claimWindowOpenBlockHash,
		supplierOperatorAddr,
	), nil
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier.
func (sq *sharedQuerier) GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetEarliestSupplierProofCommitHeight")

	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParamsUpdates, queryHeight)

	blockHashCacheKey := getBlockHashCacheKey(proofWindowOpenHeight)
	proofWindowOpenBlockHash, found := sq.blockHashCache.Get(blockHashCacheKey)

	if !found {
		// Use mutex for cache miss pattern
		sq.blockHashMutex.Lock()
		defer sq.blockHashMutex.Unlock()

		// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
		proofWindowOpenBlockHash, found = sq.blockHashCache.Get(blockHashCacheKey)
		if found {
			logger.Debug().Msgf("cache HIT for blockHeight after lock: %s", blockHashCacheKey)
		} else {
			logger.Debug().Msgf("cache MISS for blockHeight: %s", blockHashCacheKey)

			proofWindowOpenBlock, err := retry.Call(ctx, func() (*cometrpctypes.ResultBlock, error) {
				return sq.blockQuerier.Block(ctx, &proofWindowOpenHeight)
			}, retry.GetStrategy(ctx))
			if err != nil {
				return 0, err
			}

			// Cache the block hash for future use.
			proofWindowOpenBlockHash = proofWindowOpenBlock.BlockID.Hash.Bytes()
			sq.blockHashCache.Set(blockHashCacheKey, proofWindowOpenBlockHash)
		}
	} else {
		logger.Debug().Msgf("cache HIT for blockHeight: %s", blockHashCacheKey)
	}

	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParamsUpdates,
		queryHeight,
		proofWindowOpenBlockHash,
		supplierOperatorAddr,
	), nil
}

// GetComputeUnitsToTokensMultiplier returns the multiplier used to convert compute units to tokens.
func (sq *sharedQuerier) GetComputeUnitsToTokensMultiplier(
	ctx context.Context,
	queryHeight int64,
) (uint64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedParams.GetComputeUnitsToTokensMultiplier(), nil
}

func (sq *sharedQuerier) GetParamsUpdates(ctx context.Context) ([]*sharedtypes.ParamsUpdate, error) {
	cacheValueVersions, found := sq.paramsCache.GetAllUpdates()
	if !found {
		return sq.populateParamsCache(ctx)
	}

	latestVersions := cacheValueVersions.GetSortedDescVersions()
	if latestVersions == nil {
		return sq.populateParamsCache(ctx)
	}

	return sq.buildParamsUpdatesFromCache(cacheValueVersions), nil
}

// getBlockHashCacheKey constructs the cache key for a block hash by string formatting the block height.
func getBlockHashCacheKey(height int64) string {
	return strconv.FormatInt(height, 10)
}

func (sq *sharedQuerier) populateParamsCache(ctx context.Context) ([]*sharedtypes.ParamsUpdate, error) {
	response, err := sq.sharedQuerier.ParamsUpdates(ctx, &sharedtypes.QueryParamsUpdatesRequest{})
	if err != nil {
		return nil, err
	}

	for _, paramsUpdate := range response.ParamsUpdates {
		sq.paramsCache.SetAtHeight(paramsUpdate.Params, int64(paramsUpdate.EffectiveBlockHeight))
	}

	return response.ParamsUpdates, nil
}

func (sq *sharedQuerier) buildParamsUpdatesFromCache(
	cacheValueVersions cache.CacheValueHistory[sharedtypes.Params],
) []*sharedtypes.ParamsUpdate {
	latestVersions := cacheValueVersions.GetSortedDescVersions()
	versionToValueMap := cacheValueVersions.GetVersionToValueMap()

	paramsUpdate := make([]*sharedtypes.ParamsUpdate, 0, len(latestVersions))
	for i := len(latestVersions) - 1; i >= 0; i-- {
		version := latestVersions[i]
		cacheValue := versionToValueMap[version]
		paramsUpdate = append(paramsUpdate, &sharedtypes.ParamsUpdate{
			Params:               cacheValue.Value(),
			EffectiveBlockHeight: uint64(version),
		})
	}

	return paramsUpdate
}
