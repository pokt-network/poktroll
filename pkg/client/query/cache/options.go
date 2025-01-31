package cache

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// ClearableCache is an interface that defines the common methods for a cache object.
type Cache interface {
	Clear()
}

// CacheOption is a function type for the option functions that can customize
// the cache behavior.
type CacheOption[C Cache] func(context.Context, depinject.Config, C) error

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
