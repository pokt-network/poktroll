package cache

import (
	"time"
)

// EvictionPolicy determines which items are removed when number of keys in the cache reaches maxKeys.
type EvictionPolicy int64

const (
	FirstInFirstOut = EvictionPolicy(iota)
	LeastRecentlyUsed
	LeastFrequentlyUsed
)

// queryCacheConfig is the configuration for query caches.
// It is intended to be configured via QueryCacheOptionFn functions.
type queryCacheConfig struct {
	// maxKeys is the maximum number of items (key/value pairs) the cache can
	// hold before it starts evicting.
	maxKeys int64

	// TODO_CONSIDERATION:
	//
	// maxValueSize is the maximum cumulative size of all values in the cache.
	// maxValueSize int64
	// maxCacheSize is the maximum cumulative size of all keys AND values in the cache.
	// maxCacheSize int64

	// evictionPolicy determines which items are removed when number of keys in the cache reaches maxKeys.
	evictionPolicy EvictionPolicy
	// ttl is how long items should remain in the cache. Items older than the ttl
	// MAY NOT be evicted immediately, but are NEVER considered as cache hits.
	ttl time.Duration
	// historical determines whether each key will point to a single values (false)
	// or a history (i.e. reverse chronological list) of values (true).
	historical bool
	// numHistoricalValues is the number of past blocks for which to keep historical
	// values. If 0, no historical pruning is performed. It only applies when
	// historical is true.
	numHistoricalValues int64
}

// QueryCacheOptionFn is a function which receives a queryCacheConfig for configuration.
type QueryCacheOptionFn func(*queryCacheConfig)

// Validate ensures that the queryCacheConfig isn't configured with incompatible options.
func (cfg *queryCacheConfig) Validate() error {
	switch cfg.evictionPolicy {
	case FirstInFirstOut:
	// TODO_IMPROVE: support LeastRecentlyUsed and LeastFrequentlyUsed policies.
	default:
		return ErrQueryCacheConfigValidation.Wrapf("eviction policy %d not imlemented", cfg.evictionPolicy)
	}

	return nil
}

// WithHistoricalMode enables historical caching with the given numHistoricalValues
// configuration; if 0, no historical pruning is performed.
func WithHistoricalMode(numHistoricalBlocks int64) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.historical = true
		cfg.numHistoricalValues = numHistoricalBlocks
	}
}

// WithMaxKeys sets the maximum number of distinct key/value pairs the cache will
// hold before evicting according to the configured eviction policy.
func WithMaxKeys(maxKeys int64) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.maxKeys = maxKeys
	}
}

// WithEvictionPolicy sets the eviction policy.
func WithEvictionPolicy(policy EvictionPolicy) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.evictionPolicy = policy
	}
}

// WithTTL sets the time-to-live for cached items. Items older than the TTL
// MAY NOT be evicted immediately, but are NEVER considered as cache hits.
func WithTTL(ttl time.Duration) QueryCacheOptionFn {
	return func(cfg *queryCacheConfig) {
		cfg.ttl = ttl
	}
}
