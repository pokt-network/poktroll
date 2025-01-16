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
	config        queryCacheConfig
	latestVersion atomic.Int64

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

// cacheValueHistory stores cachedValues by version number and maintains a sorted
// list of version numbers for which cached values exist. This list is sorted in
// descending order to improve performance characteristics by positively correlating
// index with age.
type cacheValueHistory[T any] struct {
	// sortedDescVersions is a list of the version numbers for which values are
	// cached. It is sorted in descending order.
	sortedDescVersions []int64
	// versionToValueMap is a map from a version number to the cached value at
	// that version number, if present.
	versionToValueMap map[int64]cacheValue[T]
}

// NewInMemoryCache creates a new inMemoryCache with the configuration generated
// by the given option functions.
func NewInMemoryCache[T any](opts ...QueryCacheOptionFn) (client.QueryCache[T], error) {
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
// version, which is only updated on calls to SetAsOfVersion, and therefore is not
// guaranteed to be the current version w.r.t the blockchain.
func (c *inMemoryCache[T]) Get(key string) (T, error) {
	if c.config.historical {
		return c.GetAsOfVersion(key, c.latestVersion.Load())
	}

	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()

	var zero T

	cachedValue, exists := c.values[key]
	if !exists {
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	isTTLEnabled := c.config.ttl > 0
	isCacheValueExpired := time.Since(cachedValue.cachedAt) > c.config.ttl
	if isTTLEnabled && isCacheValueExpired {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be
		// overwritten by the next call to Set(). If usage is such that values
		// aren't being subsequently set, maxKeys (if configured) will eventually
		// cause the pruning of values with expired TTLs.
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	return cachedValue.value, nil
}

// GetAsOfVersion retrieves the value from the cache with the given key, as of the
// given version. If a value is not found for that version, the value at the nearest
// previous version is returned. If the cache is not configured for historical mode,
// it returns an error.
func (c *inMemoryCache[T]) GetAsOfVersion(key string, version int64) (T, error) {
	var zero T

	if !c.config.historical {
		return zero, ErrHistoricalModeNotEnabled
	}

	c.valuesMu.RLock()
	defer c.valuesMu.RUnlock()

	valueHistory, exists := c.valueHistories[key]
	if !exists {
		return zero, ErrCacheMiss.Wrapf("key: %s", key)
	}

	var nearestCachedVersion int64 = -1
	for _, cachedVersion := range valueHistory.sortedDescVersions {
		if cachedVersion <= version {
			nearestCachedVersion = cachedVersion
			// DEV_NOTE: Since the list is sorted in descending order, once we
			// encounter a cachedVersion that is less than or equal to version,
			// all subsequent cachedVersions SHOULD also be less than or equal to
			// version.
			break
		}
	}

	if nearestCachedVersion == -1 {
		return zero, ErrCacheMiss.Wrapf("key: %s, version: %d", key, version)
	}

	value, exists := valueHistory.versionToValueMap[nearestCachedVersion]
	if !exists {
		// DEV_NOTE: This SHOULD NEVER happen. If it does, it means that the cache has been corrupted.
		return zero, ErrCacheInternal.Wrapf("failed to load historical value for key: %s, version: %d", key, version)
	}

	isTTLEnabled := c.config.ttl > 0
	isCacheValueExpired := time.Since(value.cachedAt) > c.config.ttl
	if isTTLEnabled && isCacheValueExpired {
		// DEV_NOTE: Intentionally not pruning here to improve concurrent speed;
		// otherwise, the read lock would be insufficient. The value will be pruned
		// in the subsequent call to SetAsOfVersion() after c.config.maxVersionAge
		// blocks have elapsed. If usage is such that historical values aren't being
		// subsequently set, numHistoricalBlocks (if configured) will eventually
		// cause the pruning of historical values with expired TTLs.
		return zero, ErrCacheMiss.Wrapf("key: %s, version: %d", key, version)
	}

	return value.value, nil
}

// Set adds or updates the value in the cache for the given key. If the cache is
// configured for historical mode, it will store the value at the latest **known**
// version, which is only updated on calls to SetAsOfVersion, and therefore is not
// guaranteed to be the current version w.r.t. the blockchain.
func (c *inMemoryCache[T]) Set(key string, value T) error {
	if c.config.historical {
		return ErrUnsupportedHistoricalModeOp.Wrap("inMemoryCache#Set() is not supported in historical mode")
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	isMaxKeysConfigured := c.config.maxKeys > 0
	cacheMaxKeysReached := int64(len(c.values)) >= c.config.maxKeys
	if isMaxKeysConfigured && cacheMaxKeysReached {
		if err := c.evict(); err != nil {
			return err
		}
	}

	c.values[key] = cacheValue[T]{
		value:    value,
		cachedAt: time.Now(),
	}

	return nil
}

// SetAsOfVersion adds or updates the historical value in the cache for the given key,
// and at the version number. If the cache is not configured for historical mode, it
// returns an error.
func (c *inMemoryCache[T]) SetAsOfVersion(key string, value T, version int64) error {
	if !c.config.historical {
		return ErrHistoricalModeNotEnabled
	}

	// Update c.latestVersion if the given version is newer (higher).
	latestVersion := c.latestVersion.Load()
	if version > latestVersion {
		// NB: Only update if c.latestVersion hasn't changed since we loaded it above.
		if c.latestVersion.CompareAndSwap(latestVersion, version) {
			latestVersion = version
		} else {
			// Reload the latestVersion if it did change.
			latestVersion = c.latestVersion.Load()
		}
	}

	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	valueHistory, exists := c.valueHistories[key]
	if !exists {
		versionToValueMap := make(map[int64]cacheValue[T])
		valueHistory = cacheValueHistory[T]{
			sortedDescVersions: make([]int64, 0),
			versionToValueMap:  versionToValueMap,
		}
	}

	// Update sortedDescVersions and ensure the list is sorted in descending order.
	if _, versionExists := valueHistory.versionToValueMap[version]; !versionExists {
		valueHistory.sortedDescVersions = append(valueHistory.sortedDescVersions, version)
		sort.Slice(valueHistory.sortedDescVersions, func(i, j int) bool {
			return valueHistory.sortedDescVersions[i] > valueHistory.sortedDescVersions[j]
		})
	}

	// Prune historical values for this key, where the version
	// is older than the configured maxVersionAge.
	if c.config.maxVersionAge > 0 {
		lenCachedVersions := int64(len(valueHistory.sortedDescVersions))
		for versionIdx := lenCachedVersions - 1; versionIdx >= 0; versionIdx-- {
			cachedVersion := valueHistory.sortedDescVersions[versionIdx]

			// DEV_NOTE: Since the list is sorted, and we're iterating from lowest
			// (oldest) to highest (newest) version, once we encounter a cachedVersion
			// that is newer than the configured maxVersionAge, ALL subsequent
			// heights SHOULD also be newer than the configured maxVersionAge.
			cachedVersionAge := latestVersion - cachedVersion
			if cachedVersionAge <= c.config.maxVersionAge {
				valueHistory.sortedDescVersions = valueHistory.sortedDescVersions[:versionIdx+1]
				break
			}

			delete(valueHistory.versionToValueMap, cachedVersion)
		}
	}

	valueHistory.versionToValueMap[version] = cacheValue[T]{
		value:    value,
		cachedAt: time.Now(),
	}

	c.valueHistories[key] = valueHistory

	return nil
}

// Delete removes a value from the cache.
func (c *inMemoryCache[T]) Delete(key string) {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	if c.config.historical {
		delete(c.valueHistories, key)
	} else {
		delete(c.values, key)
	}
}

// Clear removes all values from the cache.
func (c *inMemoryCache[T]) Clear() {
	c.valuesMu.Lock()
	defer c.valuesMu.Unlock()

	if c.config.historical {
		c.valueHistories = make(map[string]cacheValueHistory[T])
	} else {
		c.values = make(map[string]cacheValue[T])
	}

	c.latestVersion.Store(0)
}

// evict removes one value from the cache, to make space for a new one,
// according to the configured eviction policy
func (c *inMemoryCache[T]) evict() error {
	if c.config.historical {
		return c.evictHistorical()
	} else {
		return c.evictNonHistorical()
	}
}

// evictHistorical removes one value (and all its versions) from the cache,
// to make space for a new one, according to the configured eviction policy.
func (c *inMemoryCache[T]) evictHistorical() error {
	switch c.config.evictionPolicy {
	case FirstInFirstOut:
		var oldestKey string
		var oldestTime time.Time
		for key, valueHistory := range c.valueHistories {
			mostRecentVersion := valueHistory.sortedDescVersions[0]
			value, exists := valueHistory.versionToValueMap[mostRecentVersion]
			if !exists {
				return ErrCacheInternal.Wrapf(
					"expected value history for key %s to contain version %d but it did not 💣",
					key, mostRecentVersion,
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
