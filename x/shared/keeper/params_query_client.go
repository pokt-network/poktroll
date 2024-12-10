package keeper

import (
	"context"
	"errors"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ParamsQuerier[*sharedtypes.Params] = (*KeeperParamsQuerier[sharedtypes.Params, Keeper])(nil)

// DEV_NOTE: Can't use cosmostypes.Msg instead of any because M
// would be a pointer but GetParams() returns a value. 🙄
type paramsKeeperIface[M any] interface {
	GetParams(context.Context) M
}

// KeeperParamsQuerier provides a base implementation of ParamsQuerier for keeper-based clients
type KeeperParamsQuerier[M any, K paramsKeeperIface[M]] struct {
	keeper K
	cache  client.HistoricalQueryCache[M]
}

// NewKeeperParamsQuerier creates a new KeeperParamsQuerier instance
func NewKeeperParamsQuerier[M any, K paramsKeeperIface[M]](
	keeper K,
	opts ...cache.CacheOption,
) *KeeperParamsQuerier[M, K] {
	// Use sensible defaults for keeper-based params cache
	defaultOpts := []cache.CacheOption{
		cache.WithHistoricalMode(100), // Keep history of last 100 blocks
		cache.WithEvictionPolicy(cache.FirstInFirstOut),
	}
	opts = append(defaultOpts, opts...)

	// TODO_IMPROVE: Implement and call a goroutine that subscribes to params updates to keep the cache hot.

	return &KeeperParamsQuerier[M, K]{
		keeper: keeper,
		cache:  cache.NewInMemoryCache[M](opts...),
	}
}

// GetParams retrieves current parameters from the keeper
func (kpq *KeeperParamsQuerier[M, K]) GetParams(ctx context.Context) (*M, error) {
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
func (kpq *KeeperParamsQuerier[M, K]) GetParamsAtHeight(ctx context.Context, height int64) (*M, error) {
	// Try cache first
	cached, err := kpq.cache.GetAtHeight("params", height)
	if err == nil {
		return &cached, nil
	}
	if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
		return &cached, err
	}

	// For now, return current params as historical params are not yet implemented
	// TODO_MAINNET: Implement historical parameter querying from state
	return kpq.GetParams(ctx)
}
