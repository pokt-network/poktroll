package query

import (
	"context"
	"errors"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ client.ParamsQuerier[cosmostypes.Msg] = (*cachedParamsQuerier[cosmostypes.Msg, paramsQuerierIface[cosmostypes.Msg]])(nil)

// paramsQuerierIface is an interface which generated query clients MUST implement
// to be compatible with the cachedParamsQuerier.
// DEV_NOTE: It is mainly required due to syntactic constraints imposed by the generics
// (i.e. otherwise, P here MUST be a value type, and there's no way to express that Q
// (below) SHOULD be the concrete type of P in NewCachedParamsQuerier).
type paramsQuerierIface[P cosmostypes.Msg] interface {
	GetParams(context.Context) (P, error)
}

// NewCachedParamsQuerier creates a new params querier with the given configuration
func NewCachedParamsQuerier[P cosmostypes.Msg, Q paramsQuerierIface[P]](
	deps depinject.Config,
	queryClientConstructor func(conn gogogrpc.ClientConn) Q,
	opts ...ParamsQuerierOptionFn,
) (_ client.ParamsQuerier[P], err error) {
	cfg := DefaultParamsQuerierConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	querier := &cachedParamsQuerier[P, Q]{
		config:      cfg,
		paramsCache: cache.NewInMemoryCache[P](cfg.CacheOpts...),
	}

	if err = depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.queryClient = queryClientConstructor(querier.clientConn)

	return querier, nil
}

// TODO_IN_THIS_COMMIT: update godoc...
// cachedParamsQuerier provides common functionality for all params queriers.
// It handles parameter caching and chain querying in a generic way, where
// R is the type of the parameters and Q is the type of the query client.
type cachedParamsQuerier[P cosmostypes.Msg, Q paramsQuerierIface[P]] struct {
	clientConn  gogogrpc.ClientConn
	queryClient Q
	paramsCache client.HistoricalQueryCache[P]
	config      *ParamsQuerierConfig
}

// TODO_IN_THIS_COMMIT: update godoc...
// GetParams implements the common parameter querying with caching
func (bq *cachedParamsQuerier[P, Q]) GetParams(ctx context.Context) (P, error) {
	logger := polylog.Ctx(ctx).With(
		"querier", bq.config.ModuleName,
		"method", "GetParams",
	)

	// Check cache first
	var paramsZero P
	cached, err := bq.paramsCache.Get("params")
	switch {
	case err == nil:
		logger.Debug().Msg("cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return paramsZero, err
	}

	logger.Debug().Msg("cache miss")

	// Query chain on cache miss
	params, err := bq.queryClient.GetParams(ctx)
	if err != nil {
		if bq.config.ModuleParamError != nil {
			return paramsZero, bq.config.ModuleParamError.Wrap(err.Error())
		}
		return paramsZero, err
	}

	// Cache the result before returning
	if err = bq.paramsCache.Set("params", params); err != nil {
		return paramsZero, err
	}

	return params, nil
}

// TODO_IN_THIS_COMMIT: update godoc...
// GetParamsAtHeight returns parameters as they were at a specific height
func (bq *cachedParamsQuerier[P, Q]) GetParamsAtHeight(ctx context.Context, height int64) (P, error) {
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
