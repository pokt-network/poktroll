package query

import (
	"context"
	"fmt"
	"sync"

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
	logger polylog.Logger

	clientConn        grpc.ClientConn
	sessionQuerier    sessiontypes.QueryClient
	sharedQueryClient client.SharedQueryClient

	// sessionsCache caches sessionQueryClient.GetSession requests
	sessionsCache cache.KeyValueCache[*sessiontypes.Session]
	// sessionsMutex to protect cache access patterns for sessions
	sessionsMutex sync.Mutex

	// paramsCache caches sessionQueryClient.Params requests
	paramsCache client.ParamsCache[sessiontypes.Params]
	// paramsMutex to protect cache access patterns for params
	paramsMutex sync.Mutex
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
	sessq.logger = sessq.logger.With("query_client", "session")

	return sessq, nil
}

// GetSession returns an sessiontypes.Session struct for a given appAddress,
// serviceId and blockHeight.
//
// It implements the SessionQueryClient#GetSession function.
func (sessq *sessionQuerier) GetSession(
	ctx context.Context,
	appAddress string,
	serviceId string,
	blockHeight int64,
) (*sessiontypes.Session, error) {
	logger := sessq.logger.
		With("method", "GetSession").
		With("appAddress", appAddress).
		With("serviceId", serviceId).
		With("blockHeight", blockHeight)

	// Get the shared parameters to calculate the session start height.
	// Use the session start height as the canonical height to be used in the cache key.
	// TODO_IMPROVE(@red-0ne): Look into caching shared params.
	sharedParams, err := sessq.sharedQueryClient.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate expected session boundaries for validation
	expectedSessionStartHeight := sharedtypes.GetSessionStartHeight(sharedParams, blockHeight)

	sessionCacheKey := getSessionCacheKey(sharedParams, appAddress, serviceId, blockHeight)

	// SOLUTION 5: Expand mutex scope to prevent race conditions
	// Acquire mutex before any cache operations to ensure consistency
	sessq.sessionsMutex.Lock()
	defer sessq.sessionsMutex.Unlock()

	// Check if the session is present in the cache.
	if session, found := sessq.sessionsCache.Get(sessionCacheKey); found {
		// SOLUTION 3: Validate the cached session matches expected session boundaries
		// If there is a cached session, check if it's at the expected start height.
		// If it's not, delete the stale cache entry and fall through to fetch fresh session from chain.
		cachedSessionStartHeight := session.GetHeader().GetSessionStartBlockHeight()

		if cachedSessionStartHeight != expectedSessionStartHeight {
			logger.Warn().
				Int64("expected_start_height", expectedSessionStartHeight).
				Int64("cached_start_height", cachedSessionStartHeight).
				Str("cached_session_id", session.GetSessionId()).
				Str("cache_key", sessionCacheKey).
				Int64("query_block_height", blockHeight).
				Msg("⚠️ Session boundaries mismatch detected in cache - invalidating stale entry")

			// Delete the stale cache entry
			sessq.sessionsCache.Delete(sessionCacheKey)
			// Fall through to fetch fresh session from chain
		} else {
			// Additional validation: ensure the cached session is for the correct app and service
			if session.GetHeader().GetApplicationAddress() != appAddress ||
				session.GetHeader().GetServiceId() != serviceId {
				logger.Warn().
					Str("expected_app", appAddress).
					Str("cached_app", session.GetHeader().GetApplicationAddress()).
					Str("expected_service", serviceId).
					Str("cached_service", session.GetHeader().GetServiceId()).
					Str("cache_key", sessionCacheKey).
					Msg("⚠️ Session app/service mismatch in cache - invalidating entry")

				sessq.sessionsCache.Delete(sessionCacheKey)
				// Fall through to fetch fresh session from chain
			} else {
				logger.Debug().Msgf("cache HIT for session key (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)
				return session, nil
			}
		}
	}

	logger.Debug().Msgf("cache MISS for session key (appAddress/serviceId/sessionStartHeight): %s", sessionCacheKey)

	req := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          serviceId,
		BlockHeight:        blockHeight,
	}
	res, err := retry.Call(ctx, func() (*sessiontypes.QueryGetSessionResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return sessq.sessionQuerier.GetSession(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return nil, ErrQueryRetrieveSession.Wrapf(
			"address: %s; serviceId: %s; block height: %d; error: [%v]",
			appAddress, serviceId, blockHeight, err,
		)
	}

	// SOLUTION 3: Final validation before caching
	fetchedStartHeight := res.Session.GetHeader().GetSessionStartBlockHeight()
	fetchedEndHeight := res.Session.GetHeader().GetSessionEndBlockHeight()

	if fetchedStartHeight != expectedSessionStartHeight || fetchedEndHeight != expectedSessionEndHeight {
		logger.Error().
			Int64("expected_start_height", expectedSessionStartHeight).
			Int64("fetched_start_height", fetchedStartHeight).
			Int64("expected_end_height", expectedSessionEndHeight).
			Int64("fetched_end_height", fetchedEndHeight).
			Str("fetched_session_id", res.Session.GetSessionId()).
			Str("cache_key", sessionCacheKey).
			Int64("query_block_height", blockHeight).
			Msg("🚨 Session boundaries mismatch from chain query - possible chain state or params change issue")
		// Still cache it as this is what the chain returned, but log the discrepancy
	}

	// Cache the session using the session key.
	sessq.sessionsCache.Set(sessionCacheKey, res.Session)
	return res.Session, nil
}

// GetParams queries & returns the session module onchain parameters.
func (sessq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	logger := sessq.logger.With("method", "GetParams")

	// Check if the params are present in the cache.
	if params, found := sessq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for session params")
		return &params, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	sessq.paramsMutex.Lock()
	defer sessq.paramsMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if params, found := sessq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for session params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache MISS for session params")

	req := &sessiontypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*sessiontypes.QueryParamsResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return sessq.sessionQuerier.Params(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
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
