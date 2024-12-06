package query

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/pkg/client/query/cache"
)

// ParamsQuerierConfig holds the configuration for parameter queriers
type ParamsQuerierConfig struct {
	// CacheOpts are the options passed to create the params cache
	CacheOpts []cache.CacheOption
	// ModuleName is used for logging and error context
	ModuleName string
	// ModuleParamError is the base error type for parameter query errors
	ModuleParamError *sdkerrors.Error
	// ParamsRequest is the request type used to query params
	ParamsRequest any
}

// ParamsQuerierOptionFn defines a function that configures a ParamsQuerierConfig
type ParamsQuerierOptionFn func(*ParamsQuerierConfig)

// DefaultParamsQuerierConfig returns the default configuration for parameter queriers
func DefaultParamsQuerierConfig() *ParamsQuerierConfig {
	return &ParamsQuerierConfig{
		CacheOpts: []cache.CacheOption{
			// TODO_IN_THIS_COMMIT: extract to constants.
			cache.WithHistoricalMode(100),
			// TODO_IN_THIS_COMMIT: reconcile the fact that MaxSize doesn't apply to historical mode...
			//cache.WithMaxSize(1),
			// TODO_IN_THIS_COMMIT: extract to constants.
			cache.WithEvictionPolicy(cache.FirstInFirstOut),
		},
	}
}

// WithModuleInfo sets the module-specific information for the querier
func WithModuleInfo(moduleName string, moduleParamError *sdkerrors.Error) ParamsQuerierOptionFn {
	return func(cfg *ParamsQuerierConfig) {
		cfg.ModuleName = moduleName
		cfg.ModuleParamError = moduleParamError
	}
}

// WithParamsRequest sets the request type for querying params
func WithParamsRequest(req any) ParamsQuerierOptionFn {
	return func(cfg *ParamsQuerierConfig) {
		cfg.ParamsRequest = req
	}
}

// WithParamsCacheOptions adds cache configuration options to the params querier
func WithParamsCacheOptions(opts ...cache.CacheOption) ParamsQuerierOptionFn {
	return func(cfg *ParamsQuerierConfig) {
		cfg.CacheOpts = append(cfg.CacheOpts, opts...)
	}
}

// SharedQuerierConfig holds the configuration for the shared querier
type SharedQuerierConfig = ParamsQuerierConfig

// SharedQuerierOptionFn defines a function that configures a SharedQuerierConfig
type SharedQuerierOptionFn = ParamsQuerierOptionFn

// DefaultSharedQuerierConfig returns the default configuration
//
// TODO_IN_THIS_COMMIT: update comment... last option wins.
func DefaultSharedQuerierConfig() *SharedQuerierConfig {
	return DefaultParamsQuerierConfig()
}

// WithCacheOptions adds cache configuration options to the shared querier
func WithCacheOptions(opts ...cache.CacheOption) SharedQuerierOptionFn {
	return func(cfg *SharedQuerierConfig) {
		cfg.CacheOpts = append(cfg.CacheOpts, opts...)
	}
}
