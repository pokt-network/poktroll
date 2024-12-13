package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ParamsQuerier[*sharedtypes.Params] = (*keeperParamsQuerier[sharedtypes.Params, Keeper])(nil)

// DEV_NOTE: Can't use cosmostypes.Msg instead of any because P
// would be a pointer but GetParams() returns a value. 🙄
type paramsKeeperIface[P any] interface {
	GetParams(context.Context) P
}

// keeperParamsQuerier provides a base implementation of ParamsQuerier for keeper-based clients
type keeperParamsQuerier[P any, K paramsKeeperIface[P]] struct {
	keeper K
	cache  client.HistoricalQueryCache[P]
}

// NewKeeperParamsQuerier creates a new keeperParamsQuerier instance
func NewKeeperParamsQuerier[P any, K paramsKeeperIface[P]](
	keeper K,
	opts ...cache.QueryCacheOptionFn,
) *keeperParamsQuerier[P, K] {
	// Use sensible defaults for keeper-based params cache
	defaultOpts := []cache.QueryCacheOptionFn{
		cache.WithEvictionPolicy(cache.FirstInFirstOut),
	}
	opts = append(defaultOpts, opts...)

	// TODO_IMPROVE: Implement and call a goroutine that subscribes to params updates to keep the cache hot.

	return &keeperParamsQuerier[P, K]{
		keeper: keeper,
		cache:  cache.NewInMemoryCache[P](opts...),
	}
}

// GetParams retrieves current parameters from the keeper
func (kpq *keeperParamsQuerier[P, K]) GetParams(ctx context.Context) (*P, error) {
	// Check cache first
	cached, err := kpq.cache.Get("params")
	if err == nil {
		return &cached, nil
	}
	if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
		return &cached, err
	}

	// On cache miss, get from keeper
	params := kpq.keeper.GetParams(ctx)

	// Cache the result
	if err := kpq.cache.Set("params", params); err != nil {
		return &params, fmt.Errorf("failed to cache params: %w", err)
	}

	return &params, nil
}

// GetParamsAtHeight retrieves parameters as they were at a specific height
//
// TODO_MAINNET(@bryanchriswhite, #931): Integrate with indexer module/mixin once available.
// Currently, this method is (and MUST) NEVER called on-chain and only exists to satisfy the
// client.ParamsQuerier interface. However, it will be needed as part of #931 to support
// querying for params at historical heights, so it's short-circuited for now to always
// return an error.
func (kpq *keeperParamsQuerier[P, K]) GetParamsAtHeight(_ context.Context, _ int64) (*P, error) {
	return nil, fmt.Errorf("TODO(#931): Support on-chain historical queries")
}
