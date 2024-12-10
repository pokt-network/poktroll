package query

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/pkg/client/query/cache"
)

const (
	defaultPruneOlderThan = 100
	defaultMaxKeys        = 1000
)

// paramsQuerierConfig is the configuration for parameter queriers. It is intended
// to be configured via ParamsQuerierOptionFn functions.
type paramsQuerierConfig struct {
	// CacheOpts are the options passed to create the params cache
	CacheOpts []cache.QueryCacheOptionFn
	// ModuleName is used for logging and error context
	ModuleName string
	// ModuleParamError is the base error type for parameter query errors
	ModuleParamError *sdkerrors.Error
}

// ParamsQuerierOptionFn is a function which receives a paramsQuerierConfig for configuration.
type ParamsQuerierOptionFn func(*paramsQuerierConfig)

// DefaultParamsQuerierConfig returns the default configuration for parameter queriers
func DefaultParamsQuerierConfig() *paramsQuerierConfig {
	return &paramsQuerierConfig{
		CacheOpts: []cache.QueryCacheOptionFn{
			cache.WithHistoricalMode(defaultPruneOlderThan),
			cache.WithMaxKeys(defaultMaxKeys),
			cache.WithEvictionPolicy(cache.FirstInFirstOut),
		},
	}
}

// WithModuleInfo sets the module name and param error for the querier.
func WithModuleInfo(moduleName string, moduleParamError *sdkerrors.Error) ParamsQuerierOptionFn {
	return func(cfg *paramsQuerierConfig) {
		cfg.ModuleName = moduleName
		cfg.ModuleParamError = moduleParamError
	}
}

// WithQueryCacheOptions is used to configure the params HistoricalQueryCache.
func WithQueryCacheOptions(opts ...cache.QueryCacheOptionFn) ParamsQuerierOptionFn {
	return func(cfg *paramsQuerierConfig) {
		cfg.CacheOpts = append(cfg.CacheOpts, opts...)
	}
}
