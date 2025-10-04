package memory

import (
	"fmt"
	"time"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/puzpuzpuz/xsync/v4"
)

var _ cache.KeyValueCache[any] = (*keyValueCache[any])(nil)

// keyValueCache provides a concurrency-safe in-memory key/value cache implementation.
type keyValueCache[T any] struct {
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
	v, ok := c.values.Load(key)
	if !ok {
		return zero, false
	}
	if time.Since(v.cachedAt) > c.config.ttl {
		// Opportunistic prune (no need for atomic compute here).
		c.values.Delete(key)
		return zero, false
	}
	return v.value, true
}

// Set adds or updates the value in the cache for the given key.
func (c *keyValueCache[T]) Set(key string, value T) {
	c.values.Store(key, cacheValue[T]{value: value, cachedAt: time.Now()})

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

// evictKey removes one key/value pair from the cache, to make space for a new one,
// according to the configured eviction policy.
func (c *keyValueCache[T]) evictKey() {
	if c.config.maxKeys <= 0 || int64(c.values.Size()) <= c.config.maxKeys {
		return
	}

	now := time.Now()

	// 1) Prefer to evict any TTL-expired entry (cheap scan, remove one).
	var expiredKey string
	c.values.Range(func(k string, v cacheValue[T]) bool {
		if now.Sub(v.cachedAt) > c.config.ttl {
			expiredKey = k
			return false // found one, stop
		}
		return true
	})
	if expiredKey != "" {
		c.values.Delete(expiredKey)
		return
	}

	// 2) Fall back to configured policy.
	switch c.config.evictionPolicy {
	case FirstInFirstOut:
		// FIFO ≈ remove the oldest by cachedAt.
		var (
			oldestKey string
			oldestAt  time.Time
			found     bool
		)
		c.values.Range(func(k string, v cacheValue[T]) bool {
			if !found || v.cachedAt.Before(oldestAt) {
				oldestKey, oldestAt, found = k, v.cachedAt, true
			}
			return true
		})
		if found {
			c.values.Delete(oldestKey)
		}
	case LeastRecentlyUsed:
		// Not implemented in original; keep behavior.
		panic("LRU eviction not implemented")
	case LeastFrequentlyUsed:
		// Not implemented in original; keep behavior.
		panic("LFU eviction not implemented")
	default:
		panic(fmt.Sprintf("unsupported eviction policy: %d", c.config.evictionPolicy))
	}
}
