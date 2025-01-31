package cache

import (
	"sync"

	"github.com/pokt-network/poktroll/pkg/client/query"
)

var _ query.ParamsCache[any] = (*paramsCache[any])(nil)

// paramsCache is a simple in-memory cache implementation for query parameters.
// It does not involve key-value pairs, but only stores a single value.
type paramsCache[T any] struct {
	cacheMu sync.RWMutex
	found   bool
	value   T
}

// NewParamsCache returns a new instance of a ParamsCache.
func NewParamsCache[T any]() query.ParamsCache[T] {
	return &paramsCache[T]{}
}

// Get returns the value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *paramsCache[T]) Get() (value T, found bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	return c.value, c.found
}

// Set sets the value in the cache.
func (c *paramsCache[T]) Set(value T) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.found = true
	c.value = value
}

// Clear empties the cache.
func (c *paramsCache[T]) Clear() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.found = false
	c.value = zeroValue[T]()
}

// zeroValue is a generic helper which returns the zero value of the given type.
func zeroValue[T any]() (zero T) {
	return zero
}
