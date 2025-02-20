package memory

import (
	fmt "fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/cache"
)

var _ cache.KeyValueCache[any] = (*keyValueCache[any])(nil)

// keyValueCache provides a concurrency-safe in-memory key/value cache implementation.
type keyValueCache[T any] struct {
	config keyValueCacheConfig

	// valuesMu is used to protect values AND valueHistories from concurrent access.
	valuesMu sync.RWMutex
	// values holds the cached values in non-historical mode.
	values map[string]cacheValue[T]
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
		values: make(map[string]cacheValue[T]),
		config: config,
	}, nil
}

// Get retrieves the value from the cache with the given key. If the cache is
// configured for historical mode, it will return the value at the latest **known**
// version, which is only updated on calls to SetAsOfVersion, and therefore is not
// guaranteed to be the current version w.r.t the blockchain.
func (c *keyValueCache[T]) Get(key string) (T, bool) {
	var zero T
	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()

	cachedValue, exists := c.values[key]
	if !exists {
		return zero, false
	}

	isTTLEnabled := c.config.ttl > 0
	isCacheValueExpired := time.Since(cachedValue.cachedAt) > c.config.ttl
	if isTTLEnabled && isCacheValueExpired {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be
		// overwritten by the next call to Set(). If usage is such that values
		// aren't being subsequently set, maxKeys (if configured) will eventually
		// cause the pruning of values with expired TTLs.
		return zero, false
	}

	return cachedValue.value, true
}

// Set adds or updates the value in the cache for the given key. If the cache is
// configured for historical mode, it will store the value at the latest **known**
// version, which is only updated on calls to SetAsOfVersion, and therefore is not
// guaranteed to be the current version w.r.t. the blockchain.
func (c *keyValueCache[T]) Set(key string, value T) error {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	c.values[key] = cacheValue[T]{
		value:    value,
		cachedAt: time.Now(),
	}

	// Evict after adding the new key/value.
	c.evict()
	return nil
}

// Delete removes a value from the cache.
func (c *keyValueCache[T]) Delete(key string) {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	delete(c.values, key)
}

// Clear removes all values from the cache.
func (c *keyValueCache[T]) Clear() {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	c.values = make(map[string]cacheValue[T])
}

// evict removes one item from the cache, to make space for a new one,
// according to the configured eviction policy.
func (c *keyValueCache[T]) evict() {
	isMaxKeysConfigured := c.config.maxKeys > 0
	cacheMaxKeysReached := int64(len(c.values)) > c.config.maxKeys
	if !isMaxKeysConfigured || !cacheMaxKeysReached {
		return
	}

	switch c.config.evictionPolicy {
	case FirstInFirstOut:
		var (
			first      = true
			oldestKey  string
			oldestTime time.Time
		)
		for key, value := range c.values {
			if first || value.cachedAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = value.cachedAt
			}
			first = false
		}
		delete(c.values, oldestKey)
		return

	case LeastRecentlyUsed:
		// TODO_IMPROVE: Implement LRU eviction
		// This will require tracking access times
		panic("LRU eviction not implemented")

	case LeastFrequentlyUsed:
		// TODO_IMPROVE: Implement LFU eviction
		// This will require tracking access times
		panic("LFU eviction not implemented")

	default:
		// DEV_NOTE: This SHOULD NEVER happen, QueryCacheConfig#Validate, SHOULD prevent it.
		panic(fmt.Sprintf("unsupported eviction policy: %d", c.config.evictionPolicy))
	}
}
