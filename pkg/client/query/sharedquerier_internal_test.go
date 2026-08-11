package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// fakeSharedQueryClient is a hand-rolled sharedtypes.QueryClient that serves a fixed live
// params value and a fixed params-history value, counting the calls to each.
//
// It exists to assert which of the two the querier actually reads: a mock that returns the
// same params from both accessors (as the RelayMiner test helpers do) cannot distinguish
// them, and that is precisely how the live-params fallthrough this file guards against went
// unnoticed.
type fakeSharedQueryClient struct {
	liveParams      sharedtypes.Params
	historicalParam sharedtypes.Params

	numParamsCalls         int
	numParamsAtHeightCalls int
}

func (f *fakeSharedQueryClient) Params(
	_ context.Context,
	_ *sharedtypes.QueryParamsRequest,
	_ ...grpc.CallOption,
) (*sharedtypes.QueryParamsResponse, error) {
	f.numParamsCalls++
	return &sharedtypes.QueryParamsResponse{Params: f.liveParams}, nil
}

func (f *fakeSharedQueryClient) ParamsAtHeight(
	_ context.Context,
	_ *sharedtypes.QueryParamsAtHeightRequest,
	_ ...grpc.CallOption,
) (*sharedtypes.QueryParamsAtHeightResponse, error) {
	f.numParamsAtHeightCalls++
	return &sharedtypes.QueryParamsAtHeightResponse{Params: f.historicalParam}, nil
}

// newTestSharedQuerier builds a sharedQuerier backed by fake, with no live-params caching
// so every GetParams call is observable in the fake's counters.
func newTestSharedQuerier(fake *fakeSharedQueryClient) *sharedQuerier {
	return &sharedQuerier{
		sharedQuerier:       fake,
		logger:              polyzero.NewLogger(),
		paramsCache:         cache.NewNoOpParamsCache[sharedtypes.Params](),
		paramsAtHeightCache: make(map[int64]sharedtypes.Params),
	}
}

// newCuttmDivergenceFake returns a fake whose live params carry a DIFFERENT
// compute_units_to_tokens_multiplier than the params effective at the queried height,
// while both share the same session_grid_anchor_height.
//
// That combination is what a governance CUTTM change actually produces onchain: the new
// value is written to live immediately, its history entry is only effective at the next
// session boundary, and the grid anchor does not move because it advances only on a
// num_blocks_per_session change.
func newCuttmDivergenceFake() *fakeSharedQueryClient {
	historicalParams := sharedtypes.DefaultParams()
	historicalParams.SessionGridAnchorHeight = 100
	historicalParams.ComputeUnitsToTokensMultiplier = 42

	liveParams := historicalParams
	liveParams.ComputeUnitsToTokensMultiplier = historicalParams.ComputeUnitsToTokensMultiplier * 2

	return &fakeSharedQueryClient{
		liveParams:      liveParams,
		historicalParam: historicalParams,
	}
}

// TestSharedQuerier_GetParamsAtHeight_DoesNotServeLiveParams pins that a height at or after
// the live grid anchor is still resolved from params history rather than served from the
// live params.
//
// Regression: GetParamsAtHeight used to short-circuit to live params whenever
// session_grid_anchor_height <= queryHeight. Since the anchor only advances on a
// num_blocks_per_session change, that guard admitted every query on a chain whose last
// timing change is in the past (mainnet: anchor 831001, session starts in the 870k range),
// silently degrading this method to GetParams. The RelayMiner then priced claims and proof
// requirements under live params while the chain used the session-start epoch — a CUTTM
// decrease made the miner skip a proof the chain still required, i.e. PROOF_MISSING and a
// slashed supplier.
func TestSharedQuerier_GetParamsAtHeight_DoesNotServeLiveParams(t *testing.T) {
	ctx := context.Background()
	fake := newCuttmDivergenceFake()
	querier := newTestSharedQuerier(fake)

	// queryHeight is past the anchor, which is exactly the case the removed fast path
	// treated as "live params already describe this height".
	const queryHeight = int64(500)
	require.Greater(t, queryHeight, int64(fake.liveParams.GetSessionGridAnchorHeight()))

	params, err := querier.GetParamsAtHeight(ctx, queryHeight)
	require.NoError(t, err)

	require.Equal(t,
		fake.historicalParam.GetComputeUnitsToTokensMultiplier(),
		params.GetComputeUnitsToTokensMultiplier(),
		"must resolve the params effective at queryHeight, not the live params",
	)
	require.Equal(t, 1, fake.numParamsAtHeightCalls, "params history must actually be queried")
}

// TestSharedQuerier_GetParamsAtHeight_MemoizesPerHeight pins that repeated lookups for the
// same height cost a single RPC, and that distinct heights are not conflated.
//
// The memo is what makes always-querying-history affordable on the per-relay paths (relay
// metering, reward eligibility) that would otherwise issue a query per relay.
func TestSharedQuerier_GetParamsAtHeight_MemoizesPerHeight(t *testing.T) {
	ctx := context.Background()
	fake := newCuttmDivergenceFake()
	querier := newTestSharedQuerier(fake)

	for i := 0; i < 3; i++ {
		_, err := querier.GetParamsAtHeight(ctx, 500)
		require.NoError(t, err)
	}
	require.Equal(t, 1, fake.numParamsAtHeightCalls, "repeat lookups for one height must hit the memo")

	_, err := querier.GetParamsAtHeight(ctx, 501)
	require.NoError(t, err)
	require.Equal(t, 2, fake.numParamsAtHeightCalls, "a distinct height must not be served from another height's entry")
}

// TestSharedQuerier_GetParamsAtHeight_MemoIsBounded pins that the memo cannot grow without
// limit in a long-lived RelayMiner process.
func TestSharedQuerier_GetParamsAtHeight_MemoIsBounded(t *testing.T) {
	ctx := context.Background()
	querier := newTestSharedQuerier(newCuttmDivergenceFake())

	for height := int64(1); height <= maxParamsAtHeightCacheEntries+1; height++ {
		_, err := querier.GetParamsAtHeight(ctx, height)
		require.NoError(t, err)
	}

	require.LessOrEqual(t, len(querier.paramsAtHeightCache), maxParamsAtHeightCacheEntries)
}

// TestSharedQuerier_GetParamsAtHeight_NonPositiveHeightUsesLiveParams pins the documented
// fallback: callers with no meaningful height (0, or an unset session header) get live
// params rather than an error or a history lookup at a nonsense height.
func TestSharedQuerier_GetParamsAtHeight_NonPositiveHeightUsesLiveParams(t *testing.T) {
	ctx := context.Background()
	fake := newCuttmDivergenceFake()
	querier := newTestSharedQuerier(fake)

	for _, queryHeight := range []int64{0, -1} {
		params, err := querier.GetParamsAtHeight(ctx, queryHeight)
		require.NoError(t, err)
		require.Equal(t,
			fake.liveParams.GetComputeUnitsToTokensMultiplier(),
			params.GetComputeUnitsToTokensMultiplier(),
		)
	}

	require.Equal(t, 0, fake.numParamsAtHeightCalls, "a non-positive height must not query params history")
}
