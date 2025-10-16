package cache

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Cache is an interface that defines the common methods for a cache object.
type Cache interface {
	Clear()
}

// CacheOption is a function type for the option functions that can customize
// the cache behavior.
type CacheOption func(context.Context, depinject.Config, Cache) error

// WithNewBlockCacheClearing is a cache option that clears the cache every time
// a new block is observed.
func WithNewBlockCacheClearing[C Cache](ctx context.Context, deps depinject.Config, cache C) error {
	var blockClient client.BlockClient
	if err := depinject.Inject(deps, &blockClient); err != nil {
		return err
	}

	channel.ForEach(
		ctx,
		blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			cache.Clear()
		},
	)

	return nil
}

// WithSessionCountCacheClearFn returns a cache option that clears the cache at
// the start of every nth session.
func WithSessionCountCacheClearFn(numSessionsToClearCache uint) func(context.Context, depinject.Config, Cache) error {
	return func(ctx context.Context, deps depinject.Config, cache Cache) error {
		var logger polylog.Logger
		var blockClient client.BlockClient
		var sharedParamsCache client.ParamsCache[sharedtypes.Params]
		if err := depinject.Inject(deps, &blockClient, &sharedParamsCache, &logger); err != nil {
			return err
		}

		channel.ForEach(
			ctx,
			blockClient.CommittedBlocksSequence(ctx),
			func(ctx context.Context, block client.Block) {
				sharedParams, shouldClearCache := shouldClearCache(sharedParamsCache)
				if !shouldClearCache {
					logger.Debug().Msg("ℹ️ Shared params not found in cache. Skipping cache altogether")
					return
				}

				currentHeight := block.Height()
				currentSessionStartHeight := sharedtypes.GetSessionStartHeight(sharedParams, currentHeight)
				currentSessionNumber := sharedtypes.GetSessionNumber(sharedParams, currentHeight)

				isAtSessionStart := currentHeight == currentSessionStartHeight
				isCacheClearableSession := currentSessionNumber%int64(numSessionsToClearCache) == 0
				if isAtSessionStart && isCacheClearableSession {
					logger.Debug().Msgf(
						"🧹 Clearing cache at session number %d (start height: %d)",
						currentSessionNumber, currentSessionStartHeight,
					)
					cache.Clear()
				}
			},
		)

		return nil
	}
}

// WithClaimSettlementCacheClearFn is used to configure cache clearing at claim settlement.
//
// This timing is critical for RelayMiningDifficulty caches to ensure suppliers aren't penalized
// for using outdated difficulty values when submitting proofs that were generated at session start.
//
// Cache clearing occurs at the height where claims are settled to:
//   - Maintain stable difficulty values throughout the proof submission window
//   - Prevent suppliers from using stale cached difficulty when submitting proofs
//   - Allow fresh difficulty calculations for the next session cycle
func WithClaimSettlementCacheClearFn() func(context.Context, depinject.Config, Cache) error {
	return func(ctx context.Context, deps depinject.Config, cache Cache) error {
		var logger polylog.Logger
		var blockClient client.BlockClient
		var sharedParamsCache client.ParamsCache[sharedtypes.Params]

		// Inject dependencies
		if err := depinject.Inject(deps, &blockClient, &sharedParamsCache, &logger); err != nil {
			return err
		}

		// Open a channel to observe committed blocks and clear the cache at the right time
		channel.ForEach(
			ctx,
			blockClient.CommittedBlocksSequence(ctx),
			func(ctx context.Context, block client.Block) {
				sharedParams, shouldClearCache := shouldClearCache(sharedParamsCache)
				if !shouldClearCache {
					logger.Debug().Msg("ℹ️ Shared params not found in cache. Skipping cache altogether")
					return
				}

				// Calculate the height at which claims for the current session will be settled
				currentHeight := block.Height()
				if isAtClaimSettlementHeight(sharedParams, currentHeight) {
					logger.Debug().Msgf("🧹 Clearing cache at claim settlement height: %d", currentHeight)
					cache.Clear()
				}
			},
		)

		return nil
	}
}

// isAtClaimSettlementHeight returns true if the current height is the height at
// which claims for the current session will be settled.
func isAtClaimSettlementHeight(sharedParams *sharedtypes.Params, currentHeight int64) bool {
	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(sharedParams, currentHeight)
	sessionEndToProofWindowCloseNumBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	claimSettlementHeight := currentSessionStartHeight + sessionEndToProofWindowCloseNumBlocks
	return currentHeight == claimSettlementHeight
}

// shouldClearCache is used bye the helpers in this file to:
// 1. Determine if the cache should be cleared.
// 2. Return the shared params if they are in the cache.
//
// Why do we use the presence of shared params in the cache to determine if the cache should be cleared?
// - SharedParams are a critical signal for when to clear the cache.
// - It helps workaround cyclical dependencies between shared params querier and cache clearing.
// - Most caching operations are dependent on shared params.
func shouldClearCache(sharedParamsCache client.ParamsCache[sharedtypes.Params]) (*sharedtypes.Params, bool) {
	sharedParams, found := sharedParamsCache.Get()
	if !found {
		return nil, false
	}
	return &sharedParams, true
}
