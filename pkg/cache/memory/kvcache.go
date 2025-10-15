package memory

import (
	"fmt"
	"time"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/pokt-network/poktroll/pkg/cache"
)

var _ cache.KeyValueCache[any] = (*keyValueCache[any])(nil)

// keyValueCache provides a concurrency-safe in-memory key/value cache implementation.
type keyValueCache[T any] struct {
	// config holds the configuration for the cache.
	config keyValueCacheConfig

	// values holds the cached values.
	values *xsync.Map[string, cacheValue[T]]
}

// cacheValue wraps cached values with a cachedAt for later comparison against
// the configured TTL.
type cacheValue[T any] struct {
	value    T
	cachedAt time.Time
}

// NewKeyValueCache creates a new keyValueCache with the configuration generated
// by the given option functions.
func NewKeyValueCache[T any](opts ...KeyValueCacheOptionFn) (*keyValueCache[T], error) {
	config := DefaultKeyValueCacheConfig

	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &keyValueCache[T]{
		values: xsync.NewMap[string, cacheValue[T]](),
		config: config,
	}, nil
}

// Get retrieves the value from the cache with the given key.
func (c *keyValueCache[T]) Get(key string) (T, bool) {
	var zero T
	cachedValue, exists := c.values.Load(key)
	if !exists {
		return zero, false
	}
	isCacheValueExpired := time.Since(cachedValue.cachedAt) > c.config.ttl
	if isCacheValueExpired {
		// Opportunistically pruning because we already checked if the cache value has expired.
		c.values.Delete(key)
		return zero, false
	}
	return cachedValue.value, true
}

// Set adds or updates the value in the cache for the given key.
func (c *keyValueCache[T]) Set(key string, value T) {
	c.values.Store(key, cacheValue[T]{
		value:    value,
		cachedAt: time.Now(),
	})

	if c.config.maxKeys > 0 && int64(c.values.Size()) > c.config.maxKeys {
		c.evictKey()
	}
}

// Delete removes a value from the cache.
func (c *keyValueCache[T]) Delete(key string) {
	c.values.Delete(key)
}

// Clear removes all values from the cache.
func (c *keyValueCache[T]) Clear() {
	c.values = xsync.NewMap[string, cacheValue[T]]()
}

// evictKey removes one key/value pair from the cache to make space for a new one.
// It evicts keys based on the following policy:
// 1. Remove any expired entries (i.e. entries that have exceeded the configured TTL).
// 2. If no expired entries are found, uses the configured eviction policy to determine which key to remove.
func (c *keyValueCache[T]) evictKey() {
	// There is more space in the cache than the configured maxKeys.
	if c.config.maxKeys <= 0 || int64(c.values.Size()) <= c.config.maxKeys {
		return
	}

	// 1) Prefer to evict any TTL-expired entry (cheap scan, remove one).
	var expiredKey string
	now := time.Now()
	c.values.Range(
		func(k string, v cacheValue[T]) bool {
			if now.Sub(v.cachedAt) > c.config.ttl {
				expiredKey = k
				return false // found one expired entry, stop
			}
			return true // continue
		})
	if expiredKey != "" {
		c.values.Delete(expiredKey)
		return
	}

	// 2) Fall back to configured policy.
	switch c.config.evictionPolicy {

	// FIFO ≈ remove the oldest by cachedAt.
	case FirstInFirstOut:
		var (
			oldestKey string
			oldestAt  time.Time
			found     bool
		)
		c.values.Range(
			func(k string, v cacheValue[T]) bool {
				if !found || v.cachedAt.Before(oldestAt) {
					oldestKey, oldestAt, found = k, v.cachedAt, true
				}
				return true
			})
		if found {
			c.values.Delete(oldestKey)
		}

	// Not implemented in original; keep behavior.
	case LeastRecentlyUsed:
		panic("LRU eviction not implemented")

	// Not implemented in original; keep behavior.
	case LeastFrequentlyUsed:
		panic("LFU eviction not implemented")

	default:
		panic(fmt.Sprintf("unsupported eviction policy: %d", c.config.evictionPolicy))
	}
}
