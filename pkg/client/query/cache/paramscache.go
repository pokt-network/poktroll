package cache

import (
	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.ParamsCache[any] = (*paramsCache[any])(nil)

// singleValueCache is the key used to store the value in the cache.
const singleValueCache = ""

// paramsCache is a simple in-memory historical cache implementation for query parameters.
// It does not involve key-value pairs, but only stores a single value.
type paramsCache[T any] struct {
	historicalKeyValueCache cache.HistoricalKeyValueCache[T]
}

// NewParamsCache returns a new instance of a ParamsCache.
func NewParamsCache[T any](opts ...memory.KeyValueCacheOptionFn) (*paramsCache[T], error) {
	historicalKeyValueCache, err := memory.NewHistoricalKeyValueCache[T](opts...)
	if err != nil {
		return nil, err
	}

	return &paramsCache[T]{
		historicalKeyValueCache,
	}, nil
}

// GetLatest returns the latest value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *paramsCache[T]) GetLatest() (value T, found bool) {
	return c.historicalKeyValueCache.GetLatestVersion(singleValueCache)
}

// GetAtHeight returns the value stored in the cache at the given height.
func (c *paramsCache[T]) GetAtHeight(height int64) (value T, found bool) {
	return c.historicalKeyValueCache.GetVersionLTE(singleValueCache, height)
}

// Set stores a value in the cache at the given height.
func (c *paramsCache[T]) SetAtHeight(value T, height int64) {
	c.historicalKeyValueCache.SetVersion(singleValueCache, value, height)
}
