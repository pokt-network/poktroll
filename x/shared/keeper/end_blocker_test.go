package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestEndBlocker_PromotesAtEffectiveHeight verifies the anchored-grid promotion (#543,
// Option B): a params epoch recorded at effective height H is promoted to live exactly at
// block H — not before (off-by-one would settle the last old-epoch block with the new N and
// lose funds, spec §4.7.1) — and blocks without a pending entry are a no-op.
func TestEndBlocker_PromotesAtEffectiveHeight(t *testing.T) {
	k, ctx := testkeeper.SharedKeeper(t)

	const (
		oldN            uint64 = 4
		newN            uint64 = 13
		effectiveHeight int64  = 5
	)

	// Live = genesis epoch (anchor 1, N=4).
	liveParams := sharedtypes.DefaultParams()
	liveParams.NumBlocksPerSession = oldN
	liveParams.SessionGridAnchorHeight = 1
	liveParams.SessionNumberAtAnchor = 1
	require.NoError(t, k.SetParams(ctx, liveParams))

	// Record a new epoch effective at block 5.
	newParams := liveParams
	newParams.NumBlocksPerSession = newN
	newParams.SessionGridAnchorHeight = uint64(effectiveHeight)
	newParams.SessionNumberAtAnchor = 2
	require.NoError(t, k.SetParamsAtHeight(ctx, effectiveHeight, newParams))

	// Blocks before the effective height: no-op, live unchanged (the last old-epoch block,
	// height 4, must still see the OLD N).
	for h := int64(1); h < effectiveHeight; h++ {
		require.NoError(t, k.EndBlocker(ctx.WithBlockHeight(h)))
		require.Equal(t, oldN, k.GetParams(ctx).NumBlocksPerSession, "live N must stay old at height %d", h)
	}

	// At the effective height: promotion fires, live becomes the new epoch.
	require.NoError(t, k.EndBlocker(ctx.WithBlockHeight(effectiveHeight)))
	promoted := k.GetParams(ctx)
	require.Equal(t, newN, promoted.NumBlocksPerSession)
	require.Equal(t, uint64(effectiveHeight), promoted.SessionGridAnchorHeight)

	// After the boundary: still a no-op (idempotent, no further history entry).
	require.NoError(t, k.EndBlocker(ctx.WithBlockHeight(effectiveHeight+1)))
	require.Equal(t, newN, k.GetParams(ctx).NumBlocksPerSession)
}
