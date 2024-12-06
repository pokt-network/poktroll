package query

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

type paramsQuerier interface {
	Params(context.Context, any) (any, error)
}

// NewParamsQuerier creates a new params querier with the given configuration
func NewParamsQuerier[P cosmostypes.Msg, C paramsQuerier](
	deps depinject.Config,
	queryClientConstructor func(grpc.ClientConn) C,
	opts ...ParamsQuerierOptionFn,
) (*baseParamsQuerier[P, C], error) {
	cfg := DefaultParamsQuerierConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	querier := &baseParamsQuerier[P, C]{
		paramsCache: cache.NewInMemoryCache[P](cfg.CacheOpts...),
	}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.blockQuerier,
	); err != nil {
		return nil, err
	}

	querier.queryClient = queryClientConstructor(querier.clientConn)

	return querier, nil
}

// baseParamsQuerier provides common functionality for all params queriers.
// It handles parameter caching and chain querying in a generic way, where
// P is the type of the parameters and C is the type of the query client.
type baseParamsQuerier[P cosmostypes.Msg, C paramsQuerier] struct {
	clientConn   grpc.ClientConn
	queryClient  C
	blockQuerier client.BlockQueryClient
	paramsCache  client.HistoricalQueryCache[P]
	config       *ParamsQuerierConfig
}

// paramsQueryFn defines a function type for querying parameters from the chain
type paramsQueryFn[P any, C any] func(context.Context, C) (P, error)

// GetParams implements the common parameter querying with caching
func (bq *baseParamsQuerier[P, C]) GetParams(ctx context.Context) (P, error) {
	logger := polylog.Ctx(ctx).With(
		"querier", bq.config.ModuleName,
		"method", "GetParams",
	)

	// Check cache first
	cached, err := bq.paramsCache.Get("params")
	switch {
	case err == nil:
		logger.Debug().Msg("cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return cached, err
	}

	logger.Debug().Msg("cache miss")

	// Query chain on cache miss
	var zero P
	if bq.config.ParamsRequest == nil {
		return zero, fmt.Errorf("params request not configured")
	}

	res, err := bq.queryClient.Params(ctx, bq.config.ParamsRequest)
	if err != nil {
		if bq.config.ModuleParamError != nil {
			return zero, bq.config.ModuleParamError.Wrap(err.Error())
		}
		return zero, err
	}

	params, ok := res.(interface{ GetParams() P })
	if !ok {
		return zero, fmt.Errorf("response does not implement GetParams method")
	}

	result := params.GetParams()

	// Cache the result before returning
	if err = bq.paramsCache.Set("params", result); err != nil {
		return result, err
	}

	return result, nil
}

// GetParamsAtHeight returns parameters as they were at a specific height
func (bq *baseParamsQuerier[P, C]) GetParamsAtHeight(ctx context.Context, height int64) (P, error) {
	logger := polylog.Ctx(ctx).With(
		"querier", bq.config.ModuleName,
		"method", "GetParamsAtHeight",
		"height", height,
	)

	// Try to get from cache at specific height
	cached, err := bq.paramsCache.GetAtHeight("params", height)
	switch {
	case err == nil:
		logger.Debug().Msg("cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return cached, err
	}

	logger.Debug().Msg("cache miss")

	// TODO_MAINNET(@bryanchriswhite): Implement querying historical params from chain
	err = cache.ErrCacheMiss.Wrapf("TODO: on-chain historical data not implemented")
	logger.Error().Msgf("%s", err)

	// Meanwhile, return current params as fallback. 😬
	return bq.GetParams(ctx)
}
