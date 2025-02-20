package memory

import (
	"fmt"
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

var (
	DefaultKeyValueCacheConfig = keyValueCacheConfig{
		evictionPolicy: FirstInFirstOut,
		// TODO_MAINNET(@bryanchriswhite): Consider how we can "guarantee" good
		// alignment between the TTL and the block production rate,
		// by accessing onchain block times directly.
		ttl: time.Minute,
	}
	DefaultHistoricialKeyValueCacheConfig = historicalKeyValueCacheConfig{
		keyValueCacheConfig: DefaultKeyValueCacheConfig,
		maxVersionAge:       10,
	}
)

// keyValueCacheConfig is the configuration for constructing a keyValueCache.
// It is intended to be configured via KeyValueCacheOptionFn functions.
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
}

// historicalKeyValueCacheConfig is the configuration for constructing a historicalKeyValueCache.
// It is intended to be configured via KeyValueCacheOptionFn functions.
type historicalKeyValueCacheConfig struct {
	keyValueCacheConfig

	// maxVersionAge is the max difference between the latest known version and
	// any other version, below which value versions are retained, and above which
	// value versions are pruned.
	// E.g.: Given a latest version of 100, and a maxVersionAge of 10, then the
	// oldest version that is not pruned is 90 (100 - 10).
	// If 0, no historical pruning is performed. It ONLY applies when historical is true.
	maxVersionAge int64
}

// KeyValueCacheOptionFn is a function which receives a keyValueCacheConfig for configuration.
type KeyValueCacheOptionFn func(keyValueConfigI) error

// Validate ensures that the keyValueCacheConfig isn't configured with incompatible options.
func (cfg *keyValueCacheConfig) Validate() error {
	switch cfg.evictionPolicy {
	case FirstInFirstOut:
	// TODO_IMPROVE: support LeastRecentlyUsed and LeastFrequentlyUsed policies.
	default:
		return cache.ErrKeyValueCacheConfigValidation.Wrapf("eviction policy %d not imlemented", cfg.evictionPolicy)
	}
	return nil
}

// Validate ensures that the historicalKeyValueCacheConfig isn't configured with incompatible options.
func (cfg *historicalKeyValueCacheConfig) Validate() error {
	switch cfg.evictionPolicy {
	case FirstInFirstOut:
	// TODO_IMPROVE: support LeastRecentlyUsed and LeastFrequentlyUsed policies.
	default:
		return cache.ErrKeyValueCacheConfigValidation.Wrapf("eviction policy %d not imlemented", cfg.evictionPolicy)
	}

	if cfg.maxVersionAge < 0 {
		return cache.ErrKeyValueCacheConfigValidation.Wrapf("maxVersionAge MUST be >= 0, got: %d", cfg.maxVersionAge)
	}

	return nil
}

// WithMaxKeys sets the maximum number of distinct key/value pairs the cache will
// hold before evicting according to the configured eviction policy.
func WithMaxKeys(maxKeys int64) KeyValueCacheOptionFn {
	return func(cfg keyValueConfigI) error {
		cfg.SetMaxKeys(maxKeys)
		return nil
	}
}

// WithEvictionPolicy sets the eviction policy.
func WithEvictionPolicy(policy EvictionPolicy) KeyValueCacheOptionFn {
	return func(cfg keyValueConfigI) error {
		cfg.SetEvictionPolicy(policy)
		return nil
	}
}

// WithTTL sets the time-to-live for cached values. Values older than the TTL
// MAY NOT be evicted immediately, but are NEVER considered as cache hits.
// NOTE: TTL is ignored by the HistoricalKeyValueCache.
func WithTTL(ttl time.Duration) KeyValueCacheOptionFn {
	return func(cfg keyValueConfigI) error {
		cfg.SetTTL(ttl)
		return nil
	}
}

// WithMaxVersionAge sets the given maxVersionAge on the configuration; if 0, no historical pruning is performed.
// It can ONLY be used in the context of a HistoricalKeyValueCache.
func WithMaxVersionAge(numRetainedVersions int64) KeyValueCacheOptionFn {
	return func(cfg keyValueConfigI) error {
		histCfg, ok := cfg.(*historicalKeyValueCacheConfig)
		if !ok {
			return fmt.Errorf("unexpected cache config type, expected %T, got: %T", histCfg, cfg)
		}

		histCfg.maxVersionAge = numRetainedVersions
		return nil
	}
}

// keyValueConfigI is an interface which is implemented by the keyValueCacheConfig.
// It is also embedded in historicalKeyValueCacheConfig so that the two constructors
// can reuse these common configuration fields, while allowing extension of the config
// struct in the historical key/value (and/or other) cache(s).
type keyValueConfigI interface {
	SetMaxKeys(maxKeys int64)
	SetEvictionPolicy(policy EvictionPolicy)
	SetTTL(ttl time.Duration)
}

func (cfg *keyValueCacheConfig) SetMaxKeys(maxKeys int64) {
	cfg.maxKeys = maxKeys
}

func (cfg *keyValueCacheConfig) SetEvictionPolicy(policy EvictionPolicy) {
	cfg.evictionPolicy = policy
}

func (cfg *keyValueCacheConfig) SetTTL(ttl time.Duration) {
	cfg.ttl = ttl
}
func (cfg *keyValueCacheConfig) GetTTL() time.Duration {
	return cfg.ttl
}
