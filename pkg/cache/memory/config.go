package memory

import (
	"time"

	"github.com/pokt-network/poktroll/pkg/cache"
)

// EvictionPolicy determines which values are removed when number of keys in the cache reaches maxKeys.
type EvictionPolicy int64

const (
	FirstInFirstOut = EvictionPolicy(iota)
	LeastRecentlyUsed
	LeastFrequentlyUsed
)

var DefaultKeyValueCacheConfig = keyValueCacheConfig{
	evictionPolicy: FirstInFirstOut,
	// TODO_MAINNET(@bryanchriswhite): Consider how we can "guarantee" good
	// alignment between the TTL and the block production rate,
	// by accessing onchain block times directly.
	ttl: time.Minute,
}

// TODO_IN_THIS_COMMIT: reconcile config(s) with splitting of the cache implementations.

// keyValueCacheConfig is the configuration for constructing a keyValueCache.
// It is intended to be configured via QueryCacheOptionFn functions.
type keyValueCacheConfig struct {
	// maxKeys is the maximum number of key/value pairs the cache can
	// hold before it starts evicting.
	maxKeys int64

	// TODO_CONSIDERATION:
	//
	// maxValueSize is the maximum cumulative size of all values in the cache.
	// maxValueSize int64
	// maxCacheSize is the maximum cumulative size of all keys AND values in the cache.
	// maxCacheSize int64

	// evictionPolicy determines which values are removed when number of keys in the cache reaches maxKeys.
	evictionPolicy EvictionPolicy
	// ttl is how long values should remain valid in the cache. Items older than the
	// ttl MAY NOT be evicted immediately, but are NEVER considered as cache hits.
	ttl time.Duration
	// historical determines whether each key will point to a single values (false)
	// or a history (i.e. reverse chronological list) of values (true).
	historical bool
	// maxVersionAge is the max difference between the latest known version and
	// any other version, below which value versions are retained, and above which
	// value versions are pruned.
	// E.g.: Given a latest version of 100, and a maxVersionAge of 10, then the
	// oldest version that is not pruned is 90 (100 - 10).
	// If 0, no historical pruning is performed. It ONLY applies when historical is true.
	maxVersionAge int64
}

// QueryCacheOptionFn is a function which receives a keyValueCacheConfig for configuration.
type QueryCacheOptionFn func(*keyValueCacheConfig)

// Validate ensures that the keyValueCacheConfig isn't configured with incompatible options.
func (cfg *keyValueCacheConfig) Validate() error {
	switch cfg.evictionPolicy {
	case FirstInFirstOut:
	// TODO_IMPROVE: support LeastRecentlyUsed and LeastFrequentlyUsed policies.
	default:
		return cache.ErrQueryCacheConfigValidation.Wrapf("eviction policy %d not imlemented", cfg.evictionPolicy)
	}

	if cfg.maxVersionAge > 0 && !cfg.historical {
		return cache.ErrQueryCacheConfigValidation.Wrap("maxVersionAge > 0 requires historical mode to be enabled")
	}

	if cfg.historical && cfg.maxVersionAge < 0 {
		return cache.ErrQueryCacheConfigValidation.Wrapf("maxVersionAge MUST be >= 0, got: %d", cfg.maxVersionAge)
	}

	return nil
}

// WithHistoricalMode enables historical caching with the given maxVersionAge
// configuration; if 0, no historical pruning is performed.
func WithHistoricalMode(numRetainedVersions int64) QueryCacheOptionFn {
	return func(cfg *keyValueCacheConfig) {
		cfg.historical = true
		cfg.maxVersionAge = numRetainedVersions
	}
}

// WithMaxKeys sets the maximum number of distinct key/value pairs the cache will
// hold before evicting according to the configured eviction policy.
func WithMaxKeys(maxKeys int64) QueryCacheOptionFn {
	return func(cfg *keyValueCacheConfig) {
		cfg.maxKeys = maxKeys
	}
}

// WithEvictionPolicy sets the eviction policy.
func WithEvictionPolicy(policy EvictionPolicy) QueryCacheOptionFn {
	return func(cfg *keyValueCacheConfig) {
		cfg.evictionPolicy = policy
	}
}

// WithTTL sets the time-to-live for cached values. Values older than the TTL
// MAY NOT be evicted immediately, but are NEVER considered as cache hits.
func WithTTL(ttl time.Duration) QueryCacheOptionFn {
	return func(cfg *keyValueCacheConfig) {
		cfg.ttl = ttl
	}
}
