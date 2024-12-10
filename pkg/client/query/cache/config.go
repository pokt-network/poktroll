package cache

import (
	"time"
)

// EvictionPolicy determines how items are removed when cache is full.
type EvictionPolicy int64

const (
	FirstInFirstOut = EvictionPolicy(iota)
	LeastRecentlyUsed
	LeastFrequentlyUsed
)

// CacheConfig is the configuration options for a cache.
type CacheConfig struct {
	// MaxKeys is the maximum number of items the cache can hold.
	MaxKeys int64
	// EvictionPolicy is how items should be removed when the cache is full.
	EvictionPolicy EvictionPolicy
	// TTL is how long items should remain in the cache
	TTL time.Duration

	// historical is whether the cache will cache a single value for each key
	// (false) or whether it will cache a history of values for each key (true).
	historical bool
	// pruneOlderThan is the number of past blocks for which to keep historical
	// values. If 0, no historical pruning is performed.
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

// WithHistoricalMode enables historical caching with the given pruneOlderThan
// configuration, if 0 no historical pruning is performed.
func WithHistoricalMode(pruneOlderThan int64) CacheOption {
	return func(cfg *CacheConfig) {
		cfg.historical = true
		cfg.pruneOlderThan = pruneOlderThan
	}
}

// WithMaxKeys sets the maximum number of distinct key/value pairs the cache will
// hold before evicting according to the configured eviction policy.
func WithMaxKeys(size int64) CacheOption {
	return func(cfg *CacheConfig) {
		cfg.MaxKeys = size
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
