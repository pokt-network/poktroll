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
	clientConn        grpc.ClientConn
	sessionQuerier    sessiontypes.QueryClient
	sharedQueryClient client.SharedQueryClient
	logger            polylog.Logger

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

	// Live params only describe the session grid at or above their own anchor. Below it,
	// GetSessionStartHeight silently falls back to the GENESIS grid (sessionGridAnchor's
	// fallback), so the derived start height belongs to no real session -- and two heights
	// in two DIFFERENT real sessions can collapse onto the same cache key. GetSession then
	// returns the wrong cached session, which surfaces downstream as a session ID mismatch
	// and rejects legitimate relays. The exposure is the grace-period window right after a
	// num_blocks_per_session change, which is what the "restart the fleet after an N change"
	// runbook step has been papering over.
	//
	// Anchoring the check on SessionGridAnchorHeight is sound HERE, unlike the fast path
	// removed from GetParamsAtHeight in this same release. That one used the anchor to
	// decide whether live described a PRICING param, and pricing changes do not move the
	// anchor, so the guard passed when it should not have. This call uses params for grid
	// math ONLY -- GetSessionStartHeight reads num_blocks_per_session and the anchor and
	// nothing else -- and recordParamsHistory advances the anchor exactly when
	// num_blocks_per_session changes. So at or above the live anchor, the live grid IS the
	// grid in effect.
	//
	// Keeping the common path on live params also respects the at-height memo's contract:
	// it is keyed by height and documented for session start/end heights only, whereas this
	// method is called with the CURRENT block height on the relay hot path. Querying
	// at-height unconditionally would add an entry per block and trip the memo's
	// drop-everything eviction, degrading every other at-height caller.
	if blockHeight < int64(sharedParams.GetSessionGridAnchorHeight()) {
		paramsAtHeight, paramsErr := sessq.sharedQueryClient.GetParamsAtHeight(ctx, blockHeight)
		if paramsErr != nil {
			return nil, paramsErr
		}
		sharedParams = paramsAtHeight
	}

	sessionCacheKey := getSessionCacheKey(sharedParams, appAddress, serviceId, blockHeight)

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

	// Cache the session using the session key.
	sessq.sessionsCache.Set(sessionCacheKey, res.Session)
	return res.Session, nil
}

// GetParams queries & returns the session module onchain parameters.
func (sessq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	logger := sessq.logger.With("query_client", "session", "method", "GetParams")

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
//
// sharedParams MUST be the params governing blockHeight's grid, not necessarily live --
// see GetSession. Passing params from a different grid derives a start height that can
// alias two distinct sessions onto one key, which returns the wrong session rather than
// merely missing the cache.
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
