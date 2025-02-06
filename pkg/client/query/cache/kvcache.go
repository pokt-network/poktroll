package cache

import (
	"sync"

	"github.com/pokt-network/poktroll/pkg/client/query"
)

var _ query.KeyValueCache[any] = (*keyValueCache[any])(nil)

// keyValueCache is a simple in-memory key-value cache implementation.
// It is safe for concurrent use.
type keyValueCache[V any] struct {
	cacheMu   sync.RWMutex
	valuesMap map[string]V
}

// NewKeyValueCache returns a new instance of a KeyValueCache.
func NewKeyValueCache[T any]() query.KeyValueCache[T] {
	return &keyValueCache[T]{
		valuesMap: make(map[string]T),
	}
}

// Get returns the value for the given key.
// A boolean is returned as the second value to indicate if the key was found in the cache.
func (c *keyValueCache[V]) Get(key string) (value V, found bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	value, found = c.valuesMap[key]
	return value, found
}

// Set sets the value for the given key.
// TODO_CONSIDERATION: Add a method to set many values and indicate whether it
// is the result of a GetAll operation. This would allow us to know whether the
// cache is populated with all the possible values, so any other GetAll operation
// could be returned from the cache.
func (c *keyValueCache[V]) Set(key string, value V) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.valuesMap[key] = value
}

// Delete deletes the value for the given key.
func (c *keyValueCache[V]) Delete(key string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	delete(c.valuesMap, key)
}

// Clear empties the whole cache.
func (c *keyValueCache[V]) Clear() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	c.valuesMap = make(map[string]V)
}
