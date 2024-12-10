package cache

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
)

var (
	_ client.QueryCache[any]           = (*InMemoryCache[any])(nil)
	_ client.HistoricalQueryCache[any] = (*InMemoryCache[any])(nil)
)

// InMemoryCache provides a concurrency-safe in-memory cache implementation with
// optional historical value support.
type InMemoryCache[T any] struct {
	config       CacheConfig
	latestHeight atomic.Int64

	itemsMu sync.RWMutex
	// items type depends on historical mode:
	// | historical mode |  type                                   |
	// | --------------- | --------------------------------------- |
	// | false           | map[string]cacheItem[T]                 |
	// | true            | map[string]map[int64]heightCacheItem[T] |
	items map[string]any
}

// cacheItem wraps cached values with metadata
type cacheItem[T any] struct {
	value     T
	timestamp time.Time
}

// heightCacheItem is used when the cache is in historical mode
type heightCacheItem[T any] struct {
	value     T
	timestamp time.Time
}

// NewInMemoryCache creates a new cache with the given configuration
func NewInMemoryCache[T any](opts ...CacheOption) *InMemoryCache[T] {
	config := CacheConfig{
		EvictionPolicy: FirstInFirstOut,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &InMemoryCache[T]{
		items:  make(map[string]interface{}),
		config: config,
	}
}

// Get retrieves an item from the cache
func (c *InMemoryCache[T]) Get(key string) (T, error) {
	if c.config.historical {
		return c.GetAtHeight(key, c.latestHeight.Load())
	}

	c.itemsMu.RLock()
	defer c.itemsMu.RUnlock()

	var zero T

	item, exists := c.items[key]
	if !exists {
		return zero, ErrCacheMiss
	}

	cItem := item.(cacheItem[T])
	if c.config.TTL > 0 && time.Since(cItem.timestamp) > c.config.TTL {
		// TODO_QUESTION: should we prune here?
		return zero, ErrCacheMiss
	}

	return cItem.value, nil
}

// GetAtHeight retrieves an item from the cache at or before the specified height
func (c *InMemoryCache[T]) GetAtHeight(key string, height int64) (T, error) {
	var zero T

	if !c.config.historical {
		return zero, ErrHistoricalModeNotEnabled
	}

	c.itemsMu.RLock()
	defer c.itemsMu.RUnlock()

	heightMap, exists := c.items[key]
	if !exists {
		return zero, ErrCacheMiss
	}

	versions := heightMap.(map[int64]heightCacheItem[T])
	var nearestHeight int64 = -1
	for h := range versions {
		if h <= height && h > nearestHeight {
			nearestHeight = h
		}
	}

	if nearestHeight == -1 {
		return zero, ErrCacheMiss
	}

	item := versions[nearestHeight]
	if c.config.TTL > 0 && time.Since(item.timestamp) > c.config.TTL {
		return zero, ErrCacheMiss
	}

	return item.value, nil
}

// Set adds or updates an item in the cache
func (c *InMemoryCache[T]) Set(key string, value T) error {
	if c.config.historical {
		return c.SetAtHeight(key, value, c.latestHeight.Load())
	}

	if c.config.MaxKeys > 0 && int64(len(c.items)) >= c.config.MaxKeys {
		c.evict()
	}

	c.itemsMu.Lock()
	defer c.itemsMu.Unlock()

	c.items[key] = cacheItem[T]{
		value:     value,
		timestamp: time.Now(),
	}

	return nil
}

// SetAtHeight adds or updates an item in the cache at a specific height
func (c *InMemoryCache[T]) SetAtHeight(key string, value T, height int64) error {
	if !c.config.historical {
		return ErrHistoricalModeNotEnabled
	}

	// Update latest height if this is newer
	latestHeight := c.latestHeight.Load()
	if height > latestHeight {
		// NB: Only update if c.latestHeight hasn't changed since we loaded it above.
		c.latestHeight.CompareAndSwap(latestHeight, height)
	}

	c.itemsMu.Lock()
	defer c.itemsMu.Unlock()

	var history map[int64]heightCacheItem[T]
	if existing, exists := c.items[key]; exists {
		history = existing.(map[int64]heightCacheItem[T])
	} else {
		history = make(map[int64]heightCacheItem[T])
		c.items[key] = history
	}

	// Prune old heights if configured
	if c.config.pruneOlderThan > 0 {
		for h := range history {
			if height-h > c.config.pruneOlderThan {
				delete(history, h)
			}
		}
	}

	history[height] = heightCacheItem[T]{
		value:     value,
		timestamp: time.Now(),
	}

	return nil
}

// Delete removes an item from the cache.
func (c *InMemoryCache[T]) Delete(key string) {
	c.itemsMu.Lock()
	defer c.itemsMu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *InMemoryCache[T]) Clear() {
	c.itemsMu.Lock()
	defer c.itemsMu.Unlock()

	c.items = make(map[string]interface{})
	c.latestHeight.Store(0)
}

// evict removes one item according to the configured eviction policy
func (c *InMemoryCache[T]) evict() {
	switch c.config.EvictionPolicy {
	case FirstInFirstOut:
		var oldestKey string
		var oldestTime time.Time
		first := true

		for key, item := range c.items {
			var itemTime time.Time
			if c.config.historical {
				versions := item.(map[int64]heightCacheItem[T])
				for _, v := range versions {
					if itemTime.IsZero() || v.timestamp.Before(itemTime) {
						itemTime = v.timestamp
					}
				}
			} else {
				itemTime = item.(cacheItem[T]).timestamp
			}

			if first || itemTime.Before(oldestTime) {
				oldestKey = key
				oldestTime = itemTime
				first = false
			}
		}
		delete(c.items, oldestKey)

	case LeastRecentlyUsed:
		// TODO: Implement LRU eviction
		// This will require tracking access times
		panic("LRU eviction not implemented")

	case LeastFrequentlyUsed:
		// TODO: Implement LFU eviction
		// This will require tracking access times
		panic("LFU eviction not implemented")

	default:
		// Default to FIFO if policy not recognized
		for key := range c.items {
			delete(c.items, key)
			return
		}
	}
}
