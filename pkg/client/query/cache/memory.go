package cache

import (
	"sort"
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
	config       queryCacheConfig
	latestHeight atomic.Int64

	itemsMu sync.RWMutex
	// items type depends on historical mode:
	// | historical mode |  type                          |
	// | --------------- | ------------------------------ |
	// | false           | map[string]cacheItem[T]        |
	// | true            | map[string]cacheItemHistory[T] |
	items map[string]any
}

// cacheItem wraps cached values with metadata
type cacheItem[T any] struct {
	value     T
	timestamp time.Time
}

type cacheItemHistory[T any] struct {
	// sortedDescHeights is a list of the heights for which values are cached.
	// It is sorted in descending order.
	sortedDescHeights []int64
	itemsByHeight     map[int64]cacheItem[T]
}

// NewInMemoryCache creates a new cache with the given configuration
func NewInMemoryCache[T any](opts ...QueryCacheOptionFn) *InMemoryCache[T] {
	config := queryCacheConfig{
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

// Get retrieves the value from the cache with the given key. If the cache is
// configured for historical mode, it will return the value at the latest **known**
// height, which is only updated on calls to SetAtHeight, and therefore is not
// guaranteed to be the current height.
func (c *InMemoryCache[T]) Get(key string) (T, error) {
	if c.config.historical {
		return c.GetAtHeight(key, c.latestHeight.Load())
	}

	c.itemsMu.RLock()
	defer c.itemsMu.RUnlock()

	var zero T

	item, exists := c.items[key]
	if !exists {
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	cItem := item.(cacheItem[T])
	if c.config.TTL > 0 && time.Since(cItem.timestamp) > c.config.TTL {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be
		// overwritten by the next call to Set().
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	return cItem.value, nil
}

// GetAtHeight retrieves the value from the cache with the given key, at the given
// height. If a value is not found for that height, the value at the nearest previous
// height is returned. If the cache is not configured for historical mode, it returns
// an error.
func (c *InMemoryCache[T]) GetAtHeight(key string, getHeight int64) (T, error) {
	var zero T

	if !c.config.historical {
		return zero, ErrHistoricalModeNotEnabled
	}

	c.itemsMu.RLock()
	defer c.itemsMu.RUnlock()

	itemHistoryAny, exists := c.items[key]
	if !exists {
		return zero, ErrCacheMiss.Wrapf("key: %s, height: %d", key, getHeight)
	}

	itemHistory := itemHistoryAny.(cacheItemHistory[T])
	var nearestCachedHeight int64 = -1
	for _, cachedHeight := range itemHistory.sortedDescHeights {
		if cachedHeight <= getHeight {
			nearestCachedHeight = cachedHeight
			// DEV_NOTE: Since the list is sorted in descending order, once we
			// encounter a cachedHeight that is less than or equal to getHeight,
			// all subsequent cachedHeights SHOULD also be less than or equal to
			// getHeight.
			break
		}
	}

	if nearestCachedHeight == -1 {
		return zero, ErrCacheMiss.Wrapf("key: %s, height: %d", key, getHeight)
	}

	item := itemHistory.itemsByHeight[nearestCachedHeight]
	if c.config.TTL > 0 && time.Since(item.timestamp) > c.config.TTL {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be pruned
		// in the subsequent call to SetAtHeight() after c.config.pruneOlderThan
		// blocks have elapsed.
		return zero, ErrCacheMiss.Wrapf("key: %s, height: %d", key, getHeight)
	}

	return item.value, nil
}

// Set adds or updates the value in the cache for the given key. If the cache is
// configured for historical mode, it will store the value at the latest **known**
// height, which is only updated on calls to SetAtHeight, and therefore is not
// guaranteed to be the current height.
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

// SetAtHeight adds or updates the historical value in the cache for the given key,
// and at the given height. If the cache is not configured for historical mode, it
// returns an error.
func (c *InMemoryCache[T]) SetAtHeight(key string, value T, setHeight int64) error {
	if !c.config.historical {
		return ErrHistoricalModeNotEnabled
	}

	// Update c.latestHeight if the given setHeight is newer (higher).
	latestHeight := c.latestHeight.Load()
	if setHeight > latestHeight {
		// NB: Only update if c.latestHeight hasn't changed since we loaded it above.
		c.latestHeight.CompareAndSwap(latestHeight, setHeight)
	}

	c.itemsMu.Lock()
	defer c.itemsMu.Unlock()

	// TODO_IN_THIS_COMMIT: refactor history to be a struct which includes sortedDescHeights...
	var itemHistory cacheItemHistory[T]
	if itemHistoryAny, exists := c.items[key]; exists {
		itemHistory = itemHistoryAny.(cacheItemHistory[T])
	} else {
		itemsByHeight := make(map[int64]cacheItem[T])
		itemHistory = cacheItemHistory[T]{
			sortedDescHeights: make([]int64, 0),
			itemsByHeight:     itemsByHeight,
		}
	}

	// Update sortedDescHeights and ensure the list is sorted in descending order.
	if _, setHeightExists := itemHistory.itemsByHeight[setHeight]; !setHeightExists {
		itemHistory.sortedDescHeights = append(itemHistory.sortedDescHeights, setHeight)
		sort.Slice(itemHistory.sortedDescHeights, func(i, j int) bool {
			return itemHistory.sortedDescHeights[i] > itemHistory.sortedDescHeights[j]
		})
	}

	c.items[key] = itemHistory

	// Prune historical values for this key, where the setHeight
	// is oder than the configured pruneOlderThan.
	if c.config.pruneOlderThan > 0 {
		for heightIdx := int64(len(itemHistory.sortedDescHeights)) - 1; heightIdx >= 0; heightIdx-- {
			cachedHeight := itemHistory.sortedDescHeights[heightIdx]

			// DEV_NOTE: Since the list is sorted, and we're iterating from highest (youngest)
			// to lowest (oldest) height, once we encounter a cachedHeight that is older than the
			// configured pruneOlderThan, ALL subsequent heights SHOULD also be older than the
			// configured pruneOlderThan.
			if setHeight-cachedHeight < c.config.pruneOlderThan {
				break
			}

			delete(itemHistory.itemsByHeight, setHeight)
		}
	}

	itemHistory.itemsByHeight[setHeight] = cacheItem[T]{
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

// Clear removes all items from the cache.
func (c *InMemoryCache[T]) Clear() {
	c.itemsMu.Lock()
	defer c.itemsMu.Unlock()

	c.items = make(map[string]interface{})
	c.latestHeight.Store(0)
}

// evict removes one item from the cache, to make space for a new one,
// according to the configured eviction policy
func (c *InMemoryCache[T]) evict() {
	switch c.config.EvictionPolicy {
	case FirstInFirstOut:
		var oldestKey string
		var oldestTime time.Time
		first := true

		for key, item := range c.items {
			var itemTime time.Time
			if c.config.historical {
				itemHistory := item.(cacheItemHistory[T])
				for _, v := range itemHistory.itemsByHeight {
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
		// TODO_IMPROVE: Implement LRU eviction
		// This will require tracking access times
		panic("LRU eviction not implemented")

	case LeastFrequentlyUsed:
		// TODO_IMPROVE: Implement LFU eviction
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
