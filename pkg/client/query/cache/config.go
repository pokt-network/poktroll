package cache

import (
	"time"
)

// EvictionPolicy determines how items are removed when number of keys in the cache reaches MaxKeys.
type EvictionPolicy int64

const (
	FirstInFirstOut = EvictionPolicy(iota)
	LeastRecentlyUsed
	LeastFrequentlyUsed
)

// queryCacheConfig is the configuration for query caches.
// It is intended to be configured via QueryCacheOptionFn functions.
type queryCacheConfig struct {
	// MaxKeys is the maximum number of items (key/value pairs) the cache can
	// hold before it starts evicting.
	MaxKeys        int64
	EvictionPolicy EvictionPolicy
	// TTL is how long items should remain in the cache. Items older than the TTL
	// MAY not be evicted but SHOULD not be considered as cache hits.
	TTL time.Duration

	// historical determines whether each key will point to a single values (false)
	// or a history (i.e. reverse chronological list) of values (true).
	historical bool
	// pruneOlderThan is the number of past blocks for which to keep historical
	// values. If 0, no historical pruning is performed. It only applies when
	// historical is true.
	pruneOlderThan int64
}

// QueryCacheOptionFn is a function which receives a queryCacheConfig for configuration.
type QueryCacheOptionFn func(*queryCacheConfig)

// WithHistoricalMode enables historical caching with the given pruneOlderThan
// configuration; if 0, no historical pruning is performed.
func WithHistoricalMode(pruneOlderThan int64) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.historical = true
		cfg.pruneOlderThan = pruneOlderThan
	}
}

// WithMaxKeys sets the maximum number of distinct key/value pairs the cache will
// hold before evicting according to the configured eviction policy.
func WithMaxKeys(maxKeys int64) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.MaxKeys = maxKeys
	}
}

// WithEvictionPolicy sets the eviction policy.
func WithEvictionPolicy(policy EvictionPolicy) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.EvictionPolicy = policy
	}
}

// WithTTL sets the time-to-live for cached items. Items older than the TTL
// MAY not be evicted but SHOULD not be considered as cache hits.
func WithTTL(ttl time.Duration) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.TTL = ttl
	}
}
