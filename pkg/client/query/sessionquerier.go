package query

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
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
	// sessionsMutex to protect cache access patterns for sessions
	sessionsMutex sync.Mutex

	// eventsParamsActivationClient is used to subscribe to session module parameters updates
	eventsParamsActivationClient client.EventsParamsActivationClient
	// paramsCache caches sessionQueryClient.Params requests
	paramsCache client.ParamsCache[sessiontypes.Params]
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - polylog.Logger
// - client.EventsParamsActivationClient
// - client.SharedQueryClient
// - cache.KeyValueCache[sessiontypes.Session]
// - client.ParamsCache[sessiontypes.Params]
func NewSessionQuerier(
	ctx context.Context,
	deps depinject.Config,
) (client.SessionQueryClient, error) {
	sessq := &sessionQuerier{}

	if err := depinject.Inject(
		deps,
		&sessq.clientConn,
		&sessq.logger,
		&sessq.eventsParamsActivationClient,
		&sessq.sharedQueryClient,
		&sessq.sessionsCache,
		&sessq.paramsCache,
	); err != nil {
		return nil, err
	}

	sessq.sessionQuerier = sessiontypes.NewQueryClient(sessq.clientConn)

	// Initialize the session module cache with all existing parameters updates:
	// - Parameters are cached as historic data, eliminating the need to invalidate the cache.
	// - The UpdateParamsCache method ensures the querier starts with the current parameters history cached.
	// - Future updates are automatically cached by subscribing to the eventsParamsActivationClient observable.
	err := querycache.UpdateParamsCache(
		ctx,
		&sessiontypes.QueryParamsUpdatesRequest{},
		toSessionParamsUpdate,
		sessq.sessionQuerier,
		sessq.eventsParamsActivationClient,
		sessq.paramsCache,
	)
	if err != nil {
		return nil, err
	}

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
	sharedParamsUpdates, err := sessq.sharedQueryClient.GetParamsUpdates(ctx)
	if err != nil {
		return nil, err
	}
	sessionCacheKey := getSessionCacheKey(sharedParamsUpdates, appAddress, serviceId, blockHeight)

	// Check if the session is present in the cache.
	if session, found := sessq.sessionsCache.Get(sessionCacheKey); found {
		logger.Debug().Msgf("cache HIT for session key (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)
		return session, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	sessq.sessionsMutex.Lock()
	defer sessq.sessionsMutex.Unlock()

	// Double-check the cache after acquiring the lock
	if session, found := sessq.sessionsCache.Get(sessionCacheKey); found {
		logger.Debug().Msgf("cache HIT for session key after lock (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)
		return session, nil
	}

	logger.Debug().Msgf("cache MISS for session key (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)

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

	// Attempt to retrieve the latest parameters from the cache.
	params, found := sessq.paramsCache.GetLatest()
	if !found {
		logger.Debug().Msg("cache MISS for session params")
		return nil, fmt.Errorf("expecting session params to be found in cache")
	}

	logger.Debug().Msg("cache HIT for session params")

	return &params, nil
}

// getSessionCacheKey constructs the cache key for a session in the form of: appAddress/serviceId/sessionStartHeight.
func getSessionCacheKey(
	sharedParamsHistory sharedtypes.ParamsHistory,
	appAddress,
	serviceId string,
	blockHeight int64,
) string {
	// Using the session start height as the canonical height ensures that the cache
	// does not duplicate entries for the same session given different block heights
	// of the same session.
	sessionStartHeight := sharedParamsHistory.GetSessionStartHeight(blockHeight)
	return fmt.Sprintf("%s/%s/%d", appAddress, serviceId, sessionStartHeight)
}

func toSessionParamsUpdate(protoMessage proto.Message) (*sessiontypes.ParamsUpdate, bool) {
	if event, ok := protoMessage.(*sessiontypes.EventParamsActivated); ok {
		return &event.ParamsUpdate, true
	}

	return nil, false
}
