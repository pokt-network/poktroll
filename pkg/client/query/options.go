package query

import (
	"context"

	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	defaultPruneOlderThan = 100
	defaultMaxKeys        = 1000
)

// paramsQuerierConfig is the configuration for parameter queriers. It is intended
// to be configured via ParamsQuerierOptionFn functions.
type paramsQuerierConfig struct {
	logger polylog.Logger
	// cacheOpts are the options passed to create the params cache
	cacheOpts []cache.QueryCacheOptionFn
	// moduleName is used for logging and error context
	moduleName string
	// moduleParamError is the base error type for parameter query errors
	moduleParamError *sdkerrors.Error
}

// ParamsQuerierOptionFn is a function which receives a paramsQuerierConfig for configuration.
type ParamsQuerierOptionFn func(*paramsQuerierConfig)

// DefaultParamsQuerierConfig returns the default configuration for parameter queriers
func DefaultParamsQuerierConfig() *paramsQuerierConfig {
	return &paramsQuerierConfig{
		cacheOpts: []cache.QueryCacheOptionFn{
			cache.WithHistoricalMode(defaultPruneOlderThan),
			cache.WithMaxKeys(defaultMaxKeys),
			cache.WithEvictionPolicy(cache.FirstInFirstOut),
		},
	}
}

// WithModuleInfo sets the module name and param error for the querier.
func WithModuleInfo(ctx context.Context, moduleName string, moduleParamError *sdkerrors.Error) ParamsQuerierOptionFn {
	logger := polylog.Ctx(ctx).With(
		"module_params_querier", moduleName,
	)
	return func(cfg *paramsQuerierConfig) {
		cfg.logger = logger
		cfg.moduleName = moduleName
		cfg.moduleParamError = moduleParamError
	}
}

// WithQueryCacheOptions is used to configure the params HistoricalQueryCache.
func WithQueryCacheOptions(opts ...cache.QueryCacheOptionFn) ParamsQuerierOptionFn {
	return func(cfg *paramsQuerierConfig) {
		cfg.cacheOpts = append(cfg.cacheOpts, opts...)
	}
}

// WithLogger sets the logger for the querier.
func WithLogger(logger polylog.Logger) ParamsQuerierOptionFn {
	return func(cfg *paramsQuerierConfig) {
		cfg.logger = logger
	}
}
