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
	clientConn     grpc.ClientConn
	sessionQuerier sessiontypes.QueryClient
	sessionCache   client.QueryCache[*sessiontypes.Session]
	paramsCache    client.QueryCache[*sessiontypes.Params]
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewSessionQuerier(deps depinject.Config) (client.SessionQueryClient, error) {
	sq := &sessionQuerier{}

	// Initialize session cache with historical mode since sessions can vary by height
	sq.sessionCache = cache.NewInMemoryCache[*sessiontypes.Session](
		// TODO_IN_THIS_COMMIT: extract to a constant.
		cache.WithMaxSize(100),
		cache.WithEvictionPolicy(cache.LeastRecentlyUsed),
		// TODO_IN_THIS_COMMIT: extract to a constant.
		cache.WithTTL(time.Hour*3),
	)

	// Initialize params cache with minimal configuration since we only need latest
	sq.paramsCache = cache.NewInMemoryCache[*sessiontypes.Params](
		// TODO_IN_THIS_COMMIT: extract to a constant.
		cache.WithHistoricalMode(100),
		// TODO_IN_THIS_COMMIT: reconcile the fact that MaxSize doesn't apply to historical mode...
		//cache.WithMaxSize(1),
		cache.WithEvictionPolicy(cache.FirstInFirstOut),
		// TODO_IN_THIS_COMMIT: extract to a constant.
		cache.WithTTL(time.Hour),
	)

	if err := depinject.Inject(
		deps,
		&sq.clientConn,
	); err != nil {
		return nil, err
	}

	// TODO_IN_THIS_COMMIT: kick off a goroutine that subscribes to params updates and populates the cache.

	sq.sessionQuerier = sessiontypes.NewQueryClient(sq.clientConn)
	return sq, nil
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

// GetParams queries & returns the session module on-chain parameters.
func (sq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	logger := polylog.Ctx(ctx).With(
		"querier", "session",
		"method", "GetSession",
	)

	// Check cache first
	cached, err := sq.paramsCache.Get("params")

	switch {
	case err == nil:
		logger.Debug().Msg("cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return nil, err
	}

	logger.Debug().Msg("cache miss")

	// If not in cache, query the chain
	req := &sessiontypes.QueryParamsRequest{}
	res, err := sq.sessionQuerier.Params(ctx, req)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("%s", err)
	}

	// Cache the result before returning
	sq.paramsCache.Set("params", &res.Params)
	return &res.Params, nil
}
