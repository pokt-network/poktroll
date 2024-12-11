package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ client.SessionQueryClient = (*sessionQuerier)(nil)

// sessionQuerier is a wrapper around the sessiontypes.QueryClient that enables the
// querying of on-chain session information through a single exposed method
// which returns an sessiontypes.Session struct
type sessionQuerier struct {
	client.ParamsQuerier[*sessiontypes.Params]

	clientConn     grpc.ClientConn
	sessionQuerier sessiontypes.QueryClient
	sessionCache   client.QueryCache[*sessiontypes.Session]
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewSessionQuerier(
	deps depinject.Config,
	paramsQuerierOpts ...ParamsQuerierOptionFn,
) (client.SessionQueryClient, error) {
	paramsQuerierCfg := DefaultParamsQuerierConfig()
	for _, opt := range paramsQuerierOpts {
		opt(paramsQuerierCfg)
	}

	paramsQuerier, err := NewCachedParamsQuerier[*sessiontypes.Params, sessiontypes.SessionQueryClient](
		deps, sessiontypes.NewSessionQueryClient,
		WithModuleInfo[*sessiontypes.Params](sessiontypes.ModuleName, sessiontypes.ErrSessionParamInvalid),
		WithParamsCacheOptions(paramsQuerierCfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	// Initialize session cache with historical mode since sessions can vary by height
	// TODO_IN_THIS_COMMIT: consider supporting multiple cache configs per query client.
	sessionCache := cache.NewInMemoryCache[*sessiontypes.Session](
		// TODO_IN_THIS_COMMIT: extract to an option fn.
		cache.WithMaxKeys(100),
		cache.WithEvictionPolicy(cache.LeastRecentlyUsed),
		// TODO_IN_THIS_COMMIT: extract to a constant.
		cache.WithTTL(time.Hour*3),
	)

	querier := &sessionQuerier{
		ParamsQuerier: paramsQuerier,
		sessionCache:  sessionCache,
	}

	if err = depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.sessionQuerier = sessiontypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetSession returns an sessiontypes.Session struct for a given appAddress,
// serviceId and blockHeight. It implements the SessionQueryClient#GetSession function.
func (sq *sessionQuerier) GetSession(
	ctx context.Context,
	appAddress string,
	serviceId string,
	blockHeight int64,
) (*sessiontypes.Session, error) {
	logger := polylog.Ctx(ctx).With(
		"querier", "session",
		"method", "GetSession",
	)

	// Create cache key from query parameters
	cacheKey := fmt.Sprintf("%s:%s:%d", appAddress, serviceId, blockHeight)

	// Check cache first
	cached, err := sq.sessionCache.Get(cacheKey)
	switch {
	case err == nil:
		logger.Debug().Msg("cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return nil, err
	default:
		logger.Debug().Msg("cache miss")
	}

	// If not cached, query the chain
	req := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          serviceId,
		BlockHeight:        blockHeight,
	}
	res, err := sq.sessionQuerier.GetSession(ctx, req)
	if err != nil {
		return nil, ErrQueryRetrieveSession.Wrapf(
			"address: %s; serviceId: %s; block height: %d; error: %s",
			appAddress, serviceId, blockHeight, err,
		)
	}

	// Cache the result before returning
	if err = sq.sessionCache.Set(cacheKey, res.Session); err != nil {
		return nil, err
	}

	return res.Session, nil
}
