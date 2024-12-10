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

// abstractParamsQuerier is NOT intended to be used for anything except the
// compile-time interface compliance assertion that immediately follows.
type abstractParamsQuerier = cachedParamsQuerier[cosmostypes.Msg, paramsQuerierIface[cosmostypes.Msg]]

var _ client.ParamsQuerier[cosmostypes.Msg] = (*abstractParamsQuerier)(nil)

// paramsQuerierIface is an interface which generated query clients MUST implement
// to be compatible with the cachedParamsQuerier.
//
// DEV_NOTE: It is mainly required due to syntactic constraints imposed by the generics
// (i.e. otherwise, P here MUST be a value type, and there's no way to express that Q
// (below) SHOULD be in terms of the concrete type of P in NewCachedParamsQuerier).
type paramsQuerierIface[P cosmostypes.Msg] interface {
	GetParams(context.Context) (P, error)
}

// NewCachedParamsQuerier creates a new, generic, params querier with the given
// concrete query client constructor and the configuration which results from
// applying the given options.
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

// cachedParamsQuerier provides a generic implementation of cached param querying.
// It handles parameter caching and chain querying in a generic way, where
// P is a pointer type of the parameters, and Q is the interface type of the
// corresponding query client.
type cachedParamsQuerier[P cosmostypes.Msg, Q paramsQuerierIface[P]] struct {
	clientConn  gogogrpc.ClientConn
	queryClient Q
	paramsCache client.HistoricalQueryCache[P]
	config      *paramsQuerierConfig
}

// GetParams returns the latest cached params, if any; otherwise, it queries the
// current on-chain params and caches them.
func (bq *cachedParamsQuerier[P, Q]) GetParams(ctx context.Context) (P, error) {
	logger := polylog.Ctx(ctx).With(
		"module", bq.config.ModuleName,
		"method", "GetParams",
	)

	// Check the cache first.
	var paramsZero P
	cached, err := bq.paramsCache.Get("params")
	switch {
	case err == nil:
		logger.Debug().Msgf("params cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return paramsZero, err
	}

	logger.Debug().Msgf("%s", err)

	// Query on-chain on cache miss.
	params, err := bq.queryClient.GetParams(ctx)
	if err != nil {
		if bq.config.ModuleParamError != nil {
			return paramsZero, bq.config.ModuleParamError.Wrap(err.Error())
		}
		return paramsZero, err
	}

	// Update the cache.
	if err = bq.paramsCache.Set("params", params); err != nil {
		return paramsZero, err
	}

	return params, nil
}

// GetParamsAtHeight returns parameters as they were as of the given height, **if
// that height is present in the cache**. Otherwise, it queries the current params
// and returns them.
//
// TODO_MAINNET(@bryanchriswhite): Once on-chain historical data is available,
// update this to query for the historical params, rather than returning the
// current params, if the case of a cache miss.
func (bq *cachedParamsQuerier[P, Q]) GetParamsAtHeight(ctx context.Context, height int64) (P, error) {
	logger := polylog.Ctx(ctx).With(
		"module", bq.config.ModuleName,
		"method", "GetParamsAtHeight",
		"height", height,
	)

	// Try to get from cache at specific height
	cached, err := bq.paramsCache.GetAtHeight("params", height)
	switch {
	case err == nil:
		logger.Debug().Msg("params cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return cached, err
	}

	logger.Debug().Msgf("%s", err)

	// TODO_MAINNET(@bryanchriswhite): Implement querying historical params from chain
	err = cache.ErrCacheMiss.Wrapf("TODO: on-chain historical data not implemented")
	logger.Error().Msgf("%s", err)

	// Meanwhile, return current params as fallback. 😬
	return bq.GetParams(ctx)
}
