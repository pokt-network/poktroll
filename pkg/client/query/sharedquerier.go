package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SharedQueryClient = (*sharedQuerier)(nil)

// maxParamsAtHeightCacheEntries bounds the params-at-height memo. The RelayMiner only
// ever asks about the handful of session heights currently in flight, so this is far
// above steady-state need; it exists purely so a long-lived process cannot grow the map
// without limit. On overflow the whole map is dropped rather than evicted entry-by-entry:
// entries are cheap to re-fetch and dropping avoids tracking per-entry recency.
const maxParamsAtHeightCacheEntries = 256

// sharedQuerier is a wrapper around the sharedtypes.QueryClient that enables the
// querying of onchain shared information through a single exposed method
// which returns an sharedtypes.Session struct
type sharedQuerier struct {
	clientConn    grpc.ClientConn
	sharedQuerier sharedtypes.QueryClient
	logger        polylog.Logger

	// paramsCache caches sharedQueryClient.Params requests
	paramsCache client.ParamsCache[sharedtypes.Params]
	// paramsMutex to protect cache access patterns for params
	paramsMutex sync.Mutex

	// paramsAtHeightCache memoizes ParamsAtHeight responses, keyed by query height.
	//
	// Safe to memoize indefinitely because a params-history entry is only ever recorded
	// with an effective height in the FUTURE (recordParamsHistory writes at the next
	// session start), so for any height that has already been committed the set of
	// entries at-or-below it is frozen. Callers MUST therefore only pass past or current
	// heights — every caller today passes a session start/end height of a session that
	// has already begun.
	paramsAtHeightCache map[int64]sharedtypes.Params
	// paramsAtHeightMutex guards paramsAtHeightCache.
	paramsAtHeightMutex sync.Mutex
}

// NewSharedQuerier returns a new instance of a client.SharedQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - polylog.Logger
// - client.ParamsCache[sharedtypes.Params]
func NewSharedQuerier(deps depinject.Config) (client.SharedQueryClient, error) {
	querier := &sharedQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.logger,
		&querier.paramsCache,
	); err != nil {
		return nil, err
	}

	querier.sharedQuerier = sharedtypes.NewQueryClient(querier.clientConn)
	querier.paramsAtHeightCache = make(map[int64]sharedtypes.Params)

	return querier, nil
}

// GetParams queries & returns the shared module onchain parameters.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
func (sq *sharedQuerier) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	logger := sq.logger.With("query_client", "shared", "method", "GetParams")

	// Get the params from the cache if they exist.
	if params, found := sq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for shared params")
		return &params, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	sq.paramsMutex.Lock()
	defer sq.paramsMutex.Unlock()

	// Double-check the cache after acquiring the lock
	if params, found := sq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for shared params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache MISS for shared params")

	req := &sharedtypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*sharedtypes.QueryParamsResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return sq.sharedQuerier.Params(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	// Update the cache with the newly retrieved params.
	sq.paramsCache.Set(res.Params)
	return &res.Params, nil
}

// GetParamsAtHeight queries & returns the shared params that were effective at queryHeight.
//
// Every consumer that must agree with the chain about a specific session reads through
// here rather than through GetParams:
//
//   - PRICING (compute_units_to_tokens_multiplier, compute_unit_cost_granularity) at the
//     session START height, matching x/proof (create_claim, submit_proof,
//     ProofRequirementForClaim) and x/tokenomics settlement. Pricing under live params
//     while the chain prices at session start makes the RelayMiner skip a proof the chain
//     still requires, which surfaces as PROOF_MISSING and slashes the supplier.
//   - WINDOW TIMING (num_blocks_per_session and the claim/proof window offsets) at the
//     session END height, matching x/proof validateClaimWindow / validateProofWindow. An
//     old-epoch session measured on the live grid resolves to the wrong window and the
//     claim or proof is submitted outside it.
//
// queryHeight <= 0 falls back to the live params.
//
// DEV_NOTE — there is deliberately NO "live params already describe this height" fast path.
// A previous implementation served live params whenever
// session_grid_anchor_height <= queryHeight, on the theory that live always describes the
// currently-effective epoch (#543 Option B). That invariant is narrower than it looks:
//
//   - The anchor advances ONLY when num_blocks_per_session changes
//     (x/shared/keeper/msg_server_update_param.go), so a CUTTM or window-offset change
//     leaves it untouched and the guard admits every height.
//   - Non-timing params (the pricing pair, unbonding periods) are written to LIVE
//     immediately while their history entry is effective only at the next session
//     boundary, so live and at-height legitimately disagree for the rest of the session.
//
// Net effect: the guard passed for essentially every real query (mainnet anchor 831001 vs.
// session starts in the 870k range) and silently degraded this method to GetParams. The
// memo below replaces it — it costs one RPC per distinct session height instead of one per
// call, without assuming anything about which epoch live belongs to.
func (sq *sharedQuerier) GetParamsAtHeight(ctx context.Context, queryHeight int64) (*sharedtypes.Params, error) {
	if queryHeight <= 0 {
		return sq.GetParams(ctx)
	}

	if params, found := sq.getCachedParamsAtHeight(queryHeight); found {
		return params, nil
	}

	logger := sq.logger.With("query_client", "shared", "method", "GetParamsAtHeight")

	req := &sharedtypes.QueryParamsAtHeightRequest{Height: queryHeight}
	res, err := retry.Call(ctx, func() (*sharedtypes.QueryParamsAtHeightResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return sq.sharedQuerier.ParamsAtHeight(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	sq.setCachedParamsAtHeight(queryHeight, res.Params)

	return &res.Params, nil
}

// getCachedParamsAtHeight returns a copy of the memoized params for queryHeight, if any.
// A copy (not a pointer into the map) is returned so a caller mutating the result cannot
// corrupt the memo for every other caller.
func (sq *sharedQuerier) getCachedParamsAtHeight(queryHeight int64) (*sharedtypes.Params, bool) {
	sq.paramsAtHeightMutex.Lock()
	defer sq.paramsAtHeightMutex.Unlock()

	params, found := sq.paramsAtHeightCache[queryHeight]
	if !found {
		return nil, false
	}

	return &params, true
}

// setCachedParamsAtHeight memoizes params for queryHeight, dropping the whole memo first
// if it has grown past maxParamsAtHeightCacheEntries.
func (sq *sharedQuerier) setCachedParamsAtHeight(queryHeight int64, params sharedtypes.Params) {
	sq.paramsAtHeightMutex.Lock()
	defer sq.paramsAtHeightMutex.Unlock()

	// The nil check also covers a sharedQuerier built without NewSharedQuerier (e.g. a
	// struct literal in a test), which would otherwise panic writing to a nil map.
	if sq.paramsAtHeightCache == nil || len(sq.paramsAtHeightCache) >= maxParamsAtHeightCacheEntries {
		sq.paramsAtHeightCache = make(map[int64]sharedtypes.Params)
	}

	sq.paramsAtHeightCache[queryHeight] = params
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
// we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
//
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
// we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetSessionGracePeriodEndHeight(sharedParams, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	// Do NOT fetch the claim-window-open block hash. The claimWindowOpenBlockHash arg
	// to sharedtypes.GetEarliestSupplierClaimCommitHeight is unused (claim distribution
	// seeding is disabled), so the value is discarded. This mirrors the same removal on
	// the on-chain twin, x/proof/types/shared_query_client.go (#1976) — see the
	// CONSENSUS HARDENING note there.
	//
	// Off-chain the stakes are availability rather than consensus: fetching it cost a
	// full blockQuerier.Block RPC per claim/proof window, and — worse — turned any RPC
	// failure (pruned node, transient error) into an error return from this method for
	// a value nobody reads. Passing nil removes both.
	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		queryHeight,
		nil,
		supplierOperatorAddr,
	), nil
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@red-0ne, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierOperatorAddr string) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	// Do NOT fetch the proof-window-open block hash; see the detailed note in
	// GetEarliestSupplierClaimCommitHeight above. The block-hash arg is unused (proof
	// distribution seeding is disabled), so pass nil rather than paying an RPC — and
	// risking an error return — for a discarded value.
	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		queryHeight,
		nil,
		supplierOperatorAddr,
	), nil
}
