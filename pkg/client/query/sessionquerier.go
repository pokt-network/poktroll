package query

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SessionQueryClient = (*sessionQuerier)(nil)

// sessionQuerier is a wrapper around the sessiontypes.QueryClient that enables the
// querying of onchain session information through a single exposed method
// which returns an sessiontypes.Session struct
type sessionQuerier struct {
	clientConn        grpc.ClientConn
	sessionQuerier    sessiontypes.QueryClient
	sharedQueryClient client.SharedQueryClient
	logger            polylog.Logger

	// sessionsCache caches sessionQueryClient.GetSession requests
	sessionsCache cache.KeyValueCache[*sessiontypes.Session]
	// paramsCache caches sessionQueryClient.Params requests
	paramsCache client.ParamsCache[sessiontypes.Params]
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewSessionQuerier(deps depinject.Config) (client.SessionQueryClient, error) {
	sessq := &sessionQuerier{}

	if err := depinject.Inject(
		deps,
		&sessq.clientConn,
		&sessq.sharedQueryClient,
		&sessq.logger,
		&sessq.sessionsCache,
		&sessq.paramsCache,
	); err != nil {
		return nil, err
	}

	sessq.sessionQuerier = sessiontypes.NewQueryClient(sessq.clientConn)

	return sessq, nil
}

// GetSession returns an sessiontypes.Session struct for a given appAddress,
// serviceId and blockHeight. It implements the SessionQueryClient#GetSession function.
func (sessq *sessionQuerier) GetSession(
	ctx context.Context,
	appAddress string,
	serviceId string,
	blockHeight int64,
) (*sessiontypes.Session, error) {
	logger := sessq.logger.With("query_client", "session", "method", "GetSession")

	// Get the shared parameters to calculate the session start height.
	// Use the session start height as the canonical height to be used in the cache key.
	sharedParams, err := sessq.sharedQueryClient.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	sessionCacheKey := getSessionCacheKey(sharedParams, appAddress, serviceId, blockHeight)

	// Check if the session is present in the cache.
	if session, found := sessq.sessionsCache.Get(sessionCacheKey); found {
		logger.Debug().Msgf("cache hit for session key (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)
		return session, nil
	}

	logger.Debug().Msgf("cache miss for session key (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)

	req := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          serviceId,
		BlockHeight:        blockHeight,
	}
	res, err := retry.Call(ctx, func() (*sessiontypes.QueryGetSessionResponse, error) {
		return sessq.sessionQuerier.GetSession(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, ErrQueryRetrieveSession.Wrapf(
			"address: %s; serviceId: %s; block height: %d; error: [%v]",
			appAddress, serviceId, blockHeight, err,
		)
	}

	// Cache the session using the session key.
	sessq.sessionsCache.Set(sessionCacheKey, res.Session)
	return res.Session, nil
}

// GetParams queries & returns the session module onchain parameters.
func (sessq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	logger := sessq.logger.With("query_client", "session", "method", "GetParams")

	// Check if the params are present in the cache.
	if params, found := sessq.paramsCache.Get(); found {
		logger.Debug().Msg("cache hit for session params")
		return &params, nil
	}

	logger.Debug().Msg("cache miss for session params")

	req := &sessiontypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*sessiontypes.QueryParamsResponse, error) {
		return sessq.sessionQuerier.Params(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	// Cache the params for future queries.
	sessq.paramsCache.Set(res.Params)
	return &res.Params, nil
}

// getSessionCacheKey constructs the cache key for a session in the form of: appAddress/serviceId/sessionStartHeight.
func getSessionCacheKey(
	sharedParams *sharedtypes.Params,
	appAddress,
	serviceId string,
	blockHeight int64,
) string {
	// Using the session start height as the canonical height ensures that the cache
	// does not duplicate entries for the same session given different block heights
	// of the same session.
	sessionStartHeight := sharedtypes.GetSessionStartHeight(sharedParams, blockHeight)
	return fmt.Sprintf("%s/%s/%d", appAddress, serviceId, sessionStartHeight)
}
