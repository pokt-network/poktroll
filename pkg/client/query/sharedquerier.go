package query

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"cosmossdk.io/depinject"
	cometrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
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

	// eventsParamsActivationClient is used to subscribe to shared module parameters updates
	eventsParamsActivationClient client.EventsParamsActivationClient
	// paramsCache caches sharedQueryClient.Params requests
	paramsCache client.ParamsCache[sharedtypes.Params]
}

// NewSharedQuerier returns a new instance of a client.SharedQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - polylog.Logger
// - client.EventsParamsActivationClient
// - client.BlockQueryClient
// - cache.KeyValueCache[BlockHash]
// - client.ParamsCache[sharedtypes.Params]
func NewSharedQuerier(
	ctx context.Context,
	deps depinject.Config,
) (client.SharedQueryClient, error) {
	querier := &sharedQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.logger,
		&querier.eventsParamsActivationClient,
		&querier.blockQuerier,
		&querier.blockHashCache,
		&querier.paramsCache,
	); err != nil {
		return nil, err
	}

	querier.sharedQuerier = sharedtypes.NewQueryClient(querier.clientConn)

	// Initialize the shared module cache with all existing parameters updates:
	// - Parameters are cached as historic data, eliminating the need to invalidate the cache.
	// - The UpdateParamsCache method ensures the querier starts with the current parameters history cached.
	// - Future updates are automatically cached by subscribing to the eventsParamsActivationClient observable.
	err := querycache.UpdateParamsCache(
		ctx,
		&sharedtypes.QueryParamsUpdatesRequest{},
		toSharedParamsUpdate,
		querier.sharedQuerier,
		querier.eventsParamsActivationClient,
		querier.paramsCache,
	)
	if err != nil {
		return nil, err
	}

	return querier, nil
}

// GetParams queries & returns the shared module onchain parameters.
func (sq *sharedQuerier) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetParams")

	// Attempt to retrieve the latest parameters from the cache.
	params, found := sq.paramsCache.GetLatest()
	if !found {
		logger.Debug().Msg("cache MISS for shared params")
		return nil, fmt.Errorf("expecting shared params to be found in cache")
	}

	logger.Debug().Msg("cache HIT for shared params")

	return &params, nil
}

// GetParamsAtHeight queries & returns the shared module onchain parameters
// that were in effect at the given block height.
func (sq *sharedQuerier) GetParamsAtHeight(ctx context.Context, height int64) (*sharedtypes.Params, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetParamsAtHeight")

	// Get the params from the cache if they exist.
	params, found := sq.paramsCache.GetAtHeight(height)
	if !found {
		logger.Debug().Msgf("cache MISS for shared params at height: %d", height)
		return nil, fmt.Errorf("expecting shared params to be found in cache at height %d", height)
	}

	logger.Debug().Msgf("cache HIT for shared params at height: %d", height)

	return &params, nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
func (sq *sharedQuerier) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}
	return sharedParamsUpdates.GetClaimWindowOpenHeight(queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
func (sq *sharedQuerier) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParamsUpdates, err := sq.GetParamsUpdates(ctx)
	if err != nil {
		return 0, err
	}
	return sharedParamsUpdates.GetProofWindowOpenHeight(queryHeight), nil
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
	return sharedParamsUpdates.GetSessionGracePeriodEndHeight(queryHeight), nil
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
	claimWindowOpenHeight := sharedParamsUpdates.GetClaimWindowOpenHeight(queryHeight)

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
	proofWindowOpenHeight := sharedParamsUpdates.GetProofWindowOpenHeight(queryHeight)

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

// GetParamsUpdates returns the cached shared params history from the paramsCache.
func (sq *sharedQuerier) GetParamsUpdates(ctx context.Context) (sharedtypes.ParamsHistory, error) {
	// Params values history is expected to be cached at querier initialization.
	cacheValueVersions, found := sq.paramsCache.GetAllUpdates()
	if !found {
		return nil, fmt.Errorf("expecting shared params history to be found in cache")
	}

	latestVersions := cacheValueVersions.GetSortedDescVersions()
	versionToValueMap := cacheValueVersions.GetVersionToValueMap()

	// Reconstruct the params update history in ascending order of activation height.
	paramsUpdate := make([]*sharedtypes.ParamsUpdate, 0, len(latestVersions))
	// The latest version is at the beginning of the slice, so we iterate in reverse.
	for i := len(latestVersions) - 1; i >= 0; i-- {
		version := latestVersions[i]
		cacheValue := versionToValueMap[version]
		paramsUpdate = append(paramsUpdate, &sharedtypes.ParamsUpdate{
			Params:           cacheValue.Value(),
			ActivationHeight: version,
		})
	}

	return paramsUpdate, nil
}

// getBlockHashCacheKey constructs the cache key for a block hash by string formatting the block height.
func getBlockHashCacheKey(height int64) string {
	return strconv.FormatInt(height, 10)
}

func toSharedParamsUpdate(protoMessage proto.Message) (*sharedtypes.ParamsUpdate, bool) {
	if event, ok := protoMessage.(*sharedtypes.EventParamsActivated); ok {
		return &event.ParamsUpdate, true
	}

	return nil, false
}
