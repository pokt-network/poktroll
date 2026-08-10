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
// Window-timing computations must evaluate a session with the num_blocks_per_session that
// was in effect when that session started, not the live value. After a session-length
// change (#543 anchored grid), an old-epoch session computed with live (new-epoch) params
// would resolve to the wrong session grid and the RelayMiner would submit its claim/proof at
// the wrong window. queryHeight <= 0 falls back to the live params.
func (sq *sharedQuerier) GetParamsAtHeight(ctx context.Context, queryHeight int64) (*sharedtypes.Params, error) {
	if queryHeight <= 0 {
		return sq.GetParams(ctx)
	}

	// Fast path: if queryHeight falls within the current (live) params epoch, the live
	// params ARE the params effective at queryHeight — serve them from the existing cache
	// without an extra RPC. Under the narrow Option B invariant (#543) live params always
	// describe the currently-effective epoch, and the relayer only asks about past/current
	// session heights, so anchor <= queryHeight means queryHeight belongs to the live epoch.
	// Only an older-epoch height (queryHeight < anchor, i.e. after a recent N change) needs
	// the historical lookup.
	if liveParams, err := sq.GetParams(ctx); err == nil {
		if int64(liveParams.GetSessionGridAnchorHeight()) <= queryHeight {
			return liveParams, nil
		}
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

	return &res.Params, nil
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
