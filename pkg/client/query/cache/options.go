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

// WithSessionCountCacheClearFn returns a cache option that clears the cache at the start
// of every nth session, where n is determined by DefaultApplicationUnbondingPeriodSessions.
func WithSessionCountCacheClearFn(numSessionsToClearCache uint) func(context.Context, depinject.Config, Cache) error {
	return func(ctx context.Context, deps depinject.Config, cache Cache) error {
		var blockClient client.BlockClient
		var sharedClient client.ParamsCache[sharedtypes.Params]
		var logger polylog.Logger
		if err := depinject.Inject(deps, &blockClient, &sharedClient, &logger); err != nil {
			return err
		}

		channel.ForEach(
			ctx,
			blockClient.CommittedBlocksSequence(ctx),
			func(ctx context.Context, block client.Block) {
				sharedParams, found := sharedClient.Get()
				if !found {
					logger.Debug().Msg("ℹ️ Shared params not found in cache, skipping cache clearing")
					return
				}

				currentHeight := block.Height()
				currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, currentHeight)
				currentSessionNumber := sharedtypes.GetSessionNumber(&sharedParams, currentHeight)

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
