package cache

import (
	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.ParamsCache[any] = (*paramsCache[any])(nil)

// singleValueCache is the key used to store the value in the cache.
const singleValueCache = ""

// paramsCache is a simple in-memory cache implementation for query parameters.
// It does not involve key-value pairs, but only stores a single value.
type paramsCache[T any] struct {
	keyValueCache cache.KeyValueCache[T]
}

// NewParamsCache returns a new instance of a ParamsCache.
func NewParamsCache[T any](opts ...memory.KeyValueCacheOptionFn) (*paramsCache[T], error) {
	keyValueCache, err := memory.NewKeyValueCache[T](opts...)
	if err != nil {
		return nil, err
	}

	return &paramsCache[T]{
		keyValueCache,
	}, nil
}

// Get returns the value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *paramsCache[T]) Get() (value T, found bool) {
	return c.keyValueCache.Get(singleValueCache)
}

// Set stores a value in the cache.
func (c *paramsCache[T]) Set(value T) {
	c.keyValueCache.Set(singleValueCache, value)
}

// Delete removes the value from the cache.
func (c *paramsCache[T]) Delete() {
	c.keyValueCache.Delete(singleValueCache)
}

// Clear empties the cache.
func (c *paramsCache[T]) Clear() {
	c.keyValueCache.Delete(singleValueCache)
}
