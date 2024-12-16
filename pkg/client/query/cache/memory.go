package cache

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
)

var (
	_ client.QueryCache[any]           = (*inMemoryCache[any])(nil)
	_ client.HistoricalQueryCache[any] = (*inMemoryCache[any])(nil)

	DefaultQueryCacheConfig = queryCacheConfig{
		evictionPolicy: FirstInFirstOut,
		// TODO_MAINNET(@bryanchriswhite): Consider how we can "guarantee" good
		// alignment between the TTL and the block production rate,
		// by accessing onchain block times directly.
		ttl: time.Minute,
	}
)

// inMemoryCache provides a concurrency-safe in-memory cache implementation with
// optional historical value support.
type inMemoryCache[T any] struct {
	config       queryCacheConfig
	latestHeight atomic.Int64

	// valuesMu is used to protect values AND valueHistories from concurrent access.
	valuesMu sync.RWMutex
	// values holds the cached values in non-historical mode.
	values map[string]cacheValue[T]
	// valueHistories holds the cached historical values in historical mode.
	valueHistories map[string]cacheValueHistory[T]
}

// cacheValue wraps cached values with a cachedAt for later comparison against
// the configured TTL.
type cacheValue[T any] struct {
	value    T
	cachedAt time.Time
}

// cacheValueHistory stores cachedItems by height and maintains a sorted list of
// heights for which cached items exist. This list is sorted in descending order
// to improve performance characteristics by positively correlating index with age.
type cacheValueHistory[T any] struct {
	// sortedDescHeights is a list of the heights for which values are cached.
	// It is sorted in descending order.
	sortedDescHeights []int64
	heightMap         map[int64]cacheValue[T]
}

// NewInMemoryCache creates a new inMemoryCache with the configuration generated
// by the given option functions.
func NewInMemoryCache[T any](opts ...QueryCacheOptionFn) (*inMemoryCache[T], error) {
	config := DefaultQueryCacheConfig

	for _, opt := range opts {
		opt(&config)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &inMemoryCache[T]{
		values:         make(map[string]cacheValue[T]),
		valueHistories: make(map[string]cacheValueHistory[T]),
		config:         config,
	}, nil
}

// Get retrieves the value from the cache with the given key. If the cache is
// configured for historical mode, it will return the value at the latest **known**
// height, which is only updated on calls to SetAtHeight, and therefore is not
// guaranteed to be the current height w.r.t the blockchain.
func (c *inMemoryCache[T]) Get(key string) (T, error) {
	if c.config.historical {
		return c.GetAtHeight(key, c.latestHeight.Load())
	}

	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()

	var zero T

	cachedItem, exists := c.values[key]
	if !exists {
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	isTTLEnabled := c.config.ttl > 0
	isCacheItemExpired := time.Since(cachedItem.cachedAt) > c.config.ttl
	if isTTLEnabled && isCacheItemExpired {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be
		// overwritten by the next call to Set(). If usage is such that values
		// aren't being subsequently set, maxKeys (if configured) will eventually
		// cause the pruning of values with expired TTLs.
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	return cachedItem.value, nil
}

// GetAtHeight retrieves the value from the cache with the given key, at the given
// height. If a value is not found for that height, the value at the nearest previous
// height is returned. If the cache is not configured for historical mode, it returns
// an error.
func (c *inMemoryCache[T]) GetAtHeight(key string, getHeight int64) (T, error) {
	var zero T

	if !c.config.historical {
		return zero, ErrHistoricalModeNotEnabled
	}

	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()

	valueHistory := c.valueHistories[key]
	var nearestCachedHeight int64 = -1
	for _, cachedHeight := range valueHistory.sortedDescHeights {
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

	value, exists := valueHistory.heightMap[nearestCachedHeight]
	if !exists {
		return zero, ErrCacheInternal.Wrapf("failed to load historical value for key: %s, height: %d", key, getHeight)
	}

	if c.config.ttl > 0 && time.Since(value.cachedAt) > c.config.ttl {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be pruned
		// in the subsequent call to SetAtHeight() after c.config.numHistoricalValues
		// blocks have elapsed. If usage is such that historical values aren't being
		// subsequently set, numHistoricalBlocks (if configured) will eventually
		// cause the pruning of historical values with expired TTLs.
		return zero, ErrCacheMiss.Wrapf("key: %s, height: %d", key, getHeight)
	}

	return value.value, nil
}

// Set adds or updates the value in the cache for the given key. If the cache is
// configured for historical mode, it will store the value at the latest **known**
// height, which is only updated on calls to SetAtHeight, and therefore is not
// guaranteed to be the current height.
func (c *inMemoryCache[T]) Set(key string, value T) error {
	if c.config.historical {
		return c.SetAtHeight(key, value, c.latestHeight.Load())
	}

	isMaxKeysConfigured := c.config.maxKeys > 0
	cacheHasMaxKeys := int64(len(c.values)) >= c.config.maxKeys
	if isMaxKeysConfigured && cacheHasMaxKeys {
		if err := c.evict(); err != nil {
			return err
		}
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	c.values[key] = cacheValue[T]{
		value:    value,
		cachedAt: time.Now(),
	}

	return nil
}

// SetAtHeight adds or updates the historical value in the cache for the given key,
// and at the given height. If the cache is not configured for historical mode, it
// returns an error.
func (c *inMemoryCache[T]) SetAtHeight(key string, value T, setHeight int64) error {
	if !c.config.historical {
		return ErrHistoricalModeNotEnabled
	}

	// Update c.latestHeight if the given setHeight is newer (higher).
	latestHeight := c.latestHeight.Load()
	if setHeight > latestHeight {
		// NB: Only update if c.latestHeight hasn't changed since we loaded it above.
		c.latestHeight.CompareAndSwap(latestHeight, setHeight)
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	valueHistory, exists := c.valueHistories[key]
	if !exists {
		heightMap := make(map[int64]cacheValue[T])
		valueHistory = cacheValueHistory[T]{
			sortedDescHeights: make([]int64, 0),
			heightMap:         heightMap,
		}
	}

	// Update sortedDescHeights and ensure the list is sorted in descending order.
	if _, setHeightExists := valueHistory.heightMap[setHeight]; !setHeightExists {
		valueHistory.sortedDescHeights = append(valueHistory.sortedDescHeights, setHeight)
		sort.Slice(valueHistory.sortedDescHeights, func(i, j int) bool {
			return valueHistory.sortedDescHeights[i] > valueHistory.sortedDescHeights[j]
		})
	}

	// Prune historical values for this key, where the setHeight
	// is oder than the configured numHistoricalValues.
	if c.config.numHistoricalValues > 0 {
		lenCachedHeights := int64(len(valueHistory.sortedDescHeights))
		for heightIdx := lenCachedHeights - 1; heightIdx >= 0; heightIdx-- {
			cachedHeight := valueHistory.sortedDescHeights[heightIdx]

			// DEV_NOTE: Since the list is sorted, and we're iterating from lowest
			// (oldest) to highest (youngest) height, once we encounter a cachedHeight
			// that is younger than the configured numHistoricalValues, ALL subsequent
			// heights SHOULD also be younger than the configured numHistoricalValues.
			if setHeight-cachedHeight <= c.config.numHistoricalValues {
				valueHistory.sortedDescHeights = valueHistory.sortedDescHeights[:heightIdx+1]
				break
			}

			delete(valueHistory.heightMap, cachedHeight)
		}
	}

	valueHistory.heightMap[setHeight] = cacheValue[T]{
		value:    value,
		cachedAt: time.Now(),
	}

	c.valueHistories[key] = valueHistory

	return nil
}

// Delete removes an item from the cache.
func (c *inMemoryCache[T]) Delete(key string) {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	if c.config.historical {
		delete(c.valueHistories, key)
	} else {
		delete(c.values, key)
	}
}

// Clear removes all items from the cache.
func (c *inMemoryCache[T]) Clear() {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	if c.config.historical {
		c.valueHistories = make(map[string]cacheValueHistory[T])
	} else {
		c.values = make(map[string]cacheValue[T])
	}

	c.latestHeight.Store(0)
}

// evict removes one item from the cache, to make space for a new one,
// according to the configured eviction policy
func (c *inMemoryCache[T]) evict() error {
	if c.config.historical {
		return c.evictHistorical()
	} else {
		return c.evictNonHistorical()
	}
}

// evictHistorical removes one item from the cache, to make space for a new one,
// according to the configured eviction policy.
func (c *inMemoryCache[T]) evictHistorical() error {
	switch c.config.evictionPolicy {
	case FirstInFirstOut:
		var oldestKey string
		var oldestTime time.Time
		for key, valueHistory := range c.valueHistories {
			mostRecentHeight := valueHistory.sortedDescHeights[0]
			value, exists := valueHistory.heightMap[mostRecentHeight]
			if !exists {
				return ErrCacheInternal.Wrapf(
					"expected value history for key %s to contain height %d but it did not 💣",
					key, mostRecentHeight,
				)
			}

			if value.cachedAt.IsZero() || value.cachedAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = value.cachedAt
			}
		}
		delete(c.valueHistories, oldestKey)
		return nil

	case LeastRecentlyUsed:
		// TODO_IMPROVE: Implement LRU eviction
		// This will require tracking access times
		return ErrCacheInternal.Wrap("LRU eviction not implemented")

	case LeastFrequentlyUsed:
		// TODO_IMPROVE: Implement LFU eviction
		// This will require tracking access times
		return ErrCacheInternal.Wrap("LFU eviction not implemented")

	default:
		// DEV_NOTE: This SHOULD NEVER happen, QueryCacheConfig#Validate, SHOULD prevent it.
		return ErrCacheInternal.Wrapf("unsupported eviction policy: %d", c.config.evictionPolicy)
	}
}

// evictNonHistorical removes one item from the cache, to make space for a new one,
// according to the configured eviction policy.
func (c *inMemoryCache[T]) evictNonHistorical() error {
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
		return nil

	case LeastRecentlyUsed:
		// TODO_IMPROVE: Implement LRU eviction
		// This will require tracking access times
		return ErrCacheInternal.Wrap("LRU eviction not implemented")

	case LeastFrequentlyUsed:
		// TODO_IMPROVE: Implement LFU eviction
		// This will require tracking access times
		return ErrCacheInternal.Wrap("LFU eviction not implemented")

	default:
		// DEV_NOTE: This SHOULD NEVER happen, QueryCacheConfig#Validate, SHOULD prevent it.
		return ErrCacheInternal.Wrapf("unsupported eviction policy: %d", c.config.evictionPolicy)
	}
}
