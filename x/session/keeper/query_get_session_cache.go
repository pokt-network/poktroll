package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Compile-time check that cachedQueryServer implements types.QueryServer.
var _ types.QueryServer = (*cachedQueryServer)(nil)

// cachedQueryServer wraps a types.QueryServer with an in-memory cache for
// GetSession queries. This is ONLY used for gRPC/REST query serving, never
// for consensus-critical message handler paths.
//
// NOTE: We intentionally do NOT embed types.QueryServer as a promoted field.
// Go's devirtualization can resolve embedded interface calls to the concrete
// type at compile time, bypassing our GetSession override entirely. Instead,
// we store the inner server as a named field and explicitly delegate Params.
//
// Consensus safety: The proof module's SessionKeeper interface points directly
// to the raw Keeper (wired via depinject), completely bypassing this wrapper.
// This cache only affects the query server registered via RegisterQueryServer.
type cachedQueryServer struct {
	inner        types.QueryServer
	sessionCache cache.KeyValueCache[*types.QueryGetSessionResponse]
	keeper       Keeper
}

// Params delegates to the inner query server.
func (c *cachedQueryServer) Params(
	ctx context.Context,
	req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	return c.inner.Params(ctx, req)
}

// NewCachedQueryServer creates a query server wrapper that caches GetSession
// results. The cache is keyed by (appAddr, serviceId, sessionStartHeight) so
// that different block heights within the same session share a single entry.
//
// Memory bound: 10,000 entries * ~2-5 KB per session = ~20-50 MB max.
func (k Keeper) NewCachedQueryServer() types.QueryServer {
	sessionCache, err := memory.NewKeyValueCache[*types.QueryGetSessionResponse](
		memory.WithMaxKeys(10_000),
	)
	if err != nil {
		k.Logger().Error(fmt.Sprintf("failed to create session query cache, falling back to uncached: %v", err))
		return k
	}

	return &cachedQueryServer{
		inner:        k,
		sessionCache: sessionCache,
		keeper:       k,
	}
}

// GetSession wraps the keeper's GetSession with caching. It normalizes the
// requested block height to the session start height before checking the cache,
// ensuring that all queries within the same session window share a cache entry.
func (c *cachedQueryServer) GetSession(
	ctx context.Context,
	req *types.QueryGetSessionRequest,
) (*types.QueryGetSessionResponse, error) {
	if req == nil {
		return c.inner.GetSession(ctx, req)
	}

	// Determine the effective block height (same logic as the keeper).
	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if req.BlockHeight > 0 {
		blockHeight = req.BlockHeight
	}

	// Normalize to session start height for a stable cache key.
	sharedParams := c.keeper.sharedKeeper.GetParamsAtHeight(ctx, blockHeight)
	sessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, blockHeight)

	cacheKey := fmt.Sprintf("%s:%s:%d",
		req.ApplicationAddress, req.ServiceId, sessionStartHeight)

	// Check cache.
	if cached, found := c.sessionCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss — delegate to the real keeper.
	res, err := c.inner.GetSession(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the successful result.
	c.sessionCache.Set(cacheKey, res)
	return res, nil
}
