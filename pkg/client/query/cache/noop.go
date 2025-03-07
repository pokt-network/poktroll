package cache

import (
	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.ParamsCache[any] = (*noOpParamsCache[any])(nil)
var _ cache.KeyValueCache[any] = (*noOpKeyValueCache[any])(nil)

// noOpParamsCache is a no-op implementation of a ParamsCache.
// It does not store any values.
type noOpParamsCache[T any] struct{}

// NewParamsCache returns a new instance of a ParamsCache.
func NewNoOpParamsCache[T any]() *noOpParamsCache[T] {
	return &noOpParamsCache[T]{}
}

// Get returns the value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *noOpParamsCache[T]) Get() (value T, found bool) {
	var zeroValue T
	return zeroValue, false
}

// Set stores a value in the cache.
func (c *noOpParamsCache[T]) Set(_ T) {
}

// Delete removes the value from the cache.
func (c *noOpParamsCache[T]) Delete() {
}

// Clear empties the cache.
func (c *noOpParamsCache[T]) Clear() {
}

// noOpKeyValueCache is a no-op implementation of a KeyValueCache.
// It does not store any values.
type noOpKeyValueCache[T any] struct{}

// NewKeyValueCache returns a new instance of a KeyValueCache.
func NewNoOpKeyValueCache[T any]() *noOpKeyValueCache[T] {
	return &noOpKeyValueCache[T]{}
}

// Get returns the value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *noOpKeyValueCache[T]) Get(_ string) (value T, found bool) {
	var zeroValue T
	return zeroValue, false
}

// Set stores a value in the cache.
func (c *noOpKeyValueCache[T]) Set(_ string, _ T) {
}

// Delete removes the value from the cache.
func (c *noOpKeyValueCache[T]) Delete(_ string) {
}

// Clear empties the cache.
func (c *noOpKeyValueCache[T]) Clear() {
}
