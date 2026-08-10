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

		// Per-handler state. channel.ForEach delivers notifications sequentially on a
		// single goroutine per observer, so this needs no synchronization.
		var (
			paramsTracker            sharedParamsTracker
			lastClearedSessionNumber int64
			haveObservedBlock        bool
		)

		channel.ForEach(
			ctx,
			blockClient.CommittedBlocksSequence(ctx),
			func(ctx context.Context, block client.Block) {
				sharedParams, haveSharedParams := paramsTracker.observe(sharedParamsCache)
				if !haveSharedParams {
					logger.Debug().Msg("ℹ️ Shared params never observed. Skipping cache clear")
					return
				}

				currentHeight := block.Height()
				currentSessionNumber := sharedtypes.GetSessionNumber(sharedParams, currentHeight)

				// Adopt the in-progress session on the first block observed: the cache was
				// only just constructed, so there is nothing stale in it to clear yet.
				if !haveObservedBlock {
					haveObservedBlock = true
					lastClearedSessionNumber = currentSessionNumber
					return
				}

				// Clear once per clearable session, on the first block observed within it.
				//
				// Comparing session NUMBERS rather than matching the session start height
				// exactly means a session start block that arrives late, or is never
				// delivered at all, still produces exactly one clear for that session. The
				// previous `currentHeight == currentSessionStartHeight` test silently
				// skipped the whole session whenever its first block was missed.
				if currentSessionNumber <= lastClearedSessionNumber {
					return
				}
				if currentSessionNumber%int64(numSessionsToClearCache) != 0 {
					return
				}

				logger.Debug().Msgf(
					"🧹 Clearing cache at session number %d (start height: %d, current height: %d)",
					currentSessionNumber,
					sharedtypes.GetSessionStartHeight(sharedParams, currentHeight),
					currentHeight,
				)
				cache.Clear()
				lastClearedSessionNumber = currentSessionNumber
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

		// Per-handler state; see the note in WithSessionCountCacheClearFn.
		var paramsTracker sharedParamsTracker

		// Open a channel to observe committed blocks and clear the cache at the right time
		channel.ForEach(
			ctx,
			blockClient.CommittedBlocksSequence(ctx),
			func(ctx context.Context, block client.Block) {
				sharedParams, haveSharedParams := paramsTracker.observe(sharedParamsCache)
				if !haveSharedParams {
					logger.Debug().Msg("ℹ️ Shared params never observed. Skipping cache clear")
					return
				}

				// Calculate the height at which claims for the current session will be settled
				//
				// TODO_TECHDEBT: this is still an exact-height test, so a settlement block
				// that is never delivered skips that cycle's clear. Unlike the session-start
				// clear, missing it is the SAFE direction (difficulty stays stable for
				// longer, which is what this clear timing exists to guarantee), so it is
				// left as-is rather than made monotonic.
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
	// A session settles at sessionEnd + tail + 1 (tail = cumulative claim+proof window
	// offsets), so the session settling at currentHeight ended `tail + 1` blocks earlier.
	// The previous formula (sessionStart + tail) fired `tail` blocks into the CURRENT session
	// instead, which never happens once tail >= N (mainnet tail = 32, so N = 30/20/10 never
	// fired) and left the difficulty cache stale. Detect a real settlement boundary by
	// checking that the height `tail + 1` blocks back is an actual session end (#543, O1).
	tail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	settledSessionEndHeight := currentHeight - tail - 1
	if settledSessionEndHeight <= 0 {
		return false
	}
	return sharedtypes.IsSessionEndHeight(sharedParams, settledSessionEndHeight)
}

// sharedParamsTracker holds a clear handler's own copy of the most recently
// observed shared params.
//
// The shared params cache MUST NOT be read directly at decision time. It is itself
// registered for session-based clearing (see pkg/relayer/cmd/deps.go), so its handler
// empties it before the other handlers subscribed to the same committed-blocks
// observable get a chance to read it. Every handler that lost that race then observed
// an empty cache and skipped its own clear, and nothing repopulates the shared params
// during the fan-out because reading the cache never triggers a query.
//
// The effect was worst for the service cache: it is keyed by serviceId, so its key is
// stable while its value changes whenever a service owner updates
// compute_units_per_relay. A missed clear left the RelayMiner weighting relays with a
// stale cupr until the process restarted, and every claim it built was rejected.
//
// Keeping a local copy makes each handler's decision independent of the fan-out order.
// Shared params only change by governance, so a copy at most one block old is
// equivalent in practice, and it is strictly better than skipping the clear outright.
//
// Reading the cache (rather than querying) still avoids the cyclical dependency between
// the shared params querier and cache clearing that motivated the original gate.
type sharedParamsTracker struct {
	lastKnownSharedParams *sharedtypes.Params
}

// observe refreshes the tracker from the shared params cache and returns the most
// recent params it has seen, along with whether it has ever seen any.
func (t *sharedParamsTracker) observe(
	sharedParamsCache client.ParamsCache[sharedtypes.Params],
) (*sharedtypes.Params, bool) {
	if sharedParams, found := sharedParamsCache.Get(); found {
		t.lastKnownSharedParams = &sharedParams
	}

	return t.lastKnownSharedParams, t.lastKnownSharedParams != nil
}
