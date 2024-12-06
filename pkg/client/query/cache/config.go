package cache

import (
	"time"
)

// EvictionPolicy determines how items are removed when cache is full
type EvictionPolicy string

// TODO_IN_THIS_COMMIT: refactor to an enum.
const (
	LeastRecentlyUsed   EvictionPolicy = "LEAST_RECENTLY_USED"
	LeastFrequentlyUsed EvictionPolicy = "LEAST_FREQUENTLY_USED"
	FirstInFirstOut     EvictionPolicy = "FIRST_IN_FIRST_OUT"
)

// CacheConfig is the configuration options for a cache.
type CacheConfig struct {
	// MaxSize is the maximum number of items the cache can hold.
	MaxSize int64
	// EvictionPolicy is how items should be removed when the cache is full.
	EvictionPolicy EvictionPolicy
	// TTL is how long items should remain in the cache
	TTL time.Duration

	// historical is whether the cache will cache a single value for each key
	// (false) or whether it will cache a history of values for each key (true).
	historical bool
	// TODO_IN_THIS_COMMIT: godoc...
	pruneOlderThan int64
}

// CacheOption defines a function that configures a CacheConfig
type CacheOption func(*CacheConfig)

// HistoricalCacheConfig extends the basic CacheConfig with historical settings.
type HistoricalCacheConfig struct {
	CacheConfig

	// MaxHeightsPerKey is the maximum number of different heights to store per key
	MaxHeightsPerKey int
	// PruneOlderThan specifies how many blocks back to maintain in history
	// If 0, no historical pruning is performed
	PruneOlderThan int64
}

// WithHistoricalMode enables historical caching with the specified configuration
func WithHistoricalMode(pruneOlderThan int64) CacheOption {
	return func(cfg *CacheConfig) {
		cfg.historical = true
		cfg.pruneOlderThan = pruneOlderThan
	}
}

// WithMaxSize sets the maximum size of the cache
func WithMaxSize(size int64) CacheOption {
	return func(cfg *CacheConfig) {
		cfg.MaxSize = size
	}
}

// WithEvictionPolicy sets the eviction policy
func WithEvictionPolicy(policy EvictionPolicy) CacheOption {
	return func(cfg *CacheConfig) {
		cfg.EvictionPolicy = policy
	}
}

// WithTTL sets the time-to-live for cache entries
func WithTTL(ttl time.Duration) CacheOption {
	return func(cfg *CacheConfig) {
		cfg.TTL = ttl
	}
}
