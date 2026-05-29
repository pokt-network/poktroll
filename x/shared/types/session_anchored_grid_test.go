package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// paramsWithGrid builds a minimal shared Params carrying only the fields the anchored
// session-grid math reads: num_blocks_per_session and the grid anchor metadata.
func paramsWithGrid(numBlocksPerSession, anchor, numberAtAnchor uint64) *sharedtypes.Params {
	return &sharedtypes.Params{
		NumBlocksPerSession:     numBlocksPerSession,
		SessionGridAnchorHeight: anchor,
		SessionNumberAtAnchor:   numberAtAnchor,
	}
}

// legacyStart / legacyNumber reproduce the pre-#543 block-1 modulo grid, used to assert
// that anchor ∈ {0,1} is bit-identical to the historical behavior (spec §11.2).
func legacyStart(h, n int64) int64  { return h - ((h - 1) % n) }
func legacyNumber(h, n int64) int64 { return ((h - 1) / n) + 1 }
func legacyEnd(h, n int64) int64    { return legacyStart(h, n) + n - 1 }

func TestAnchoredGrid_LegacyEquivalence(t *testing.T) {
	const n int64 = 60
	for _, anchor := range []uint64{0, 1} {
		p := paramsWithGrid(uint64(n), anchor, anchor) // numberAtAnchor 0 or 1 both → genesis
		for h := int64(1); h <= 600; h++ {
			require.Equalf(t, legacyStart(h, n), sharedtypes.GetSessionStartHeight(p, h),
				"start mismatch at h=%d anchor=%d", h, anchor)
			require.Equalf(t, legacyEnd(h, n), sharedtypes.GetSessionEndHeight(p, h),
				"end mismatch at h=%d anchor=%d", h, anchor)
			require.Equalf(t, legacyNumber(h, n), sharedtypes.GetSessionNumber(p, h),
				"number mismatch at h=%d anchor=%d", h, anchor)
		}
	}
}

// TestAnchoredGrid_NonDivisorTransition models a 60→7 change. The first epoch is the
// genesis grid (anchor=1, N=60); the second epoch is anchored at a clean old-grid boundary
// with N=7 (a non-divisor of 60). Heights in each epoch must resolve to that epoch's grid.
func TestAnchoredGrid_NonDivisorTransition(t *testing.T) {
	// Change governed mid-session at h=130; old session [121,180] (N=60). Next boundary is 181.
	const (
		oldN   int64 = 60
		newN   int64 = 7
		anchor int64 = 181 // = old session end (180) + 1
	)
	genesisParams := paramsWithGrid(uint64(oldN), 1, 1)
	// numberAtAnchor = session number of the old session at h=130, +1.
	numAtAnchor := sharedtypes.GetSessionNumber(genesisParams, 130) + 1 // session 3 → anchor session 4
	newParams := paramsWithGrid(uint64(newN), uint64(anchor), uint64(numAtAnchor))

	// Genesis-epoch heights resolve on the old grid (unchanged).
	require.Equal(t, int64(121), sharedtypes.GetSessionStartHeight(genesisParams, 130))
	require.Equal(t, int64(180), sharedtypes.GetSessionEndHeight(genesisParams, 130))

	// New-epoch heights resolve on the 7-block grid measured from the anchor.
	require.Equal(t, anchor, sharedtypes.GetSessionStartHeight(newParams, anchor))
	require.Equal(t, anchor+newN-1, sharedtypes.GetSessionEndHeight(newParams, anchor))
	// One full session in: [anchor+7, anchor+13].
	require.Equal(t, anchor+newN, sharedtypes.GetSessionStartHeight(newParams, anchor+newN))
	require.Equal(t, anchor+2*newN-1, sharedtypes.GetSessionEndHeight(newParams, anchor+newN+3))

	// Every height within a new-epoch session shares one start/end.
	for h := anchor; h < anchor+newN; h++ {
		require.Equal(t, anchor, sharedtypes.GetSessionStartHeight(newParams, h), "h=%d", h)
		require.Equal(t, anchor+newN-1, sharedtypes.GetSessionEndHeight(newParams, h), "h=%d", h)
	}
}

// TestAnchoredGrid_MonotonicSessionNumber asserts session numbers are contiguous across an
// epoch boundary: the anchor session number is exactly old-last + 1, no reset, no gap.
func TestAnchoredGrid_MonotonicSessionNumber(t *testing.T) {
	const (
		oldN   int64 = 60
		newN   int64 = 7
		anchor int64 = 181
	)
	genesisParams := paramsWithGrid(uint64(oldN), 1, 1)
	lastOldNumber := sharedtypes.GetSessionNumber(genesisParams, anchor-1) // session at h=180
	newParams := paramsWithGrid(uint64(newN), uint64(anchor), uint64(lastOldNumber+1))

	require.Equal(t, lastOldNumber+1, sharedtypes.GetSessionNumber(newParams, anchor))
	require.Equal(t, lastOldNumber+1, sharedtypes.GetSessionNumber(newParams, anchor+newN-1))
	require.Equal(t, lastOldNumber+2, sharedtypes.GetSessionNumber(newParams, anchor+newN))
}

// TestAnchoredGrid_AnchorAfterQueryHeightGuard covers §3.4: params describing a LATER epoch
// than the query height (anchor > h) must fall back to the genesis grid, never returning a
// future/garbage start height.
func TestAnchoredGrid_AnchorAfterQueryHeightGuard(t *testing.T) {
	const n int64 = 30
	// Params for an epoch anchored at 1000, queried at height 500 (< anchor).
	p := paramsWithGrid(uint64(n), 1000, 50)

	h := int64(500)
	// Falls back to genesis block-1 grid, NOT 1000 + ((500-1000)/30)*30 (which truncates up).
	require.Equal(t, legacyStart(h, n), sharedtypes.GetSessionStartHeight(p, h))
	require.Equal(t, legacyNumber(h, n), sharedtypes.GetSessionNumber(p, h))
	require.LessOrEqual(t, sharedtypes.GetSessionStartHeight(p, h), h, "start must never exceed query height")
}

func TestAnchoredGrid_NonPositiveHeight(t *testing.T) {
	p := paramsWithGrid(60, 1, 1)
	require.Equal(t, int64(0), sharedtypes.GetSessionStartHeight(p, 0))
	require.Equal(t, int64(0), sharedtypes.GetSessionEndHeight(p, -5))
	require.Equal(t, int64(0), sharedtypes.GetSessionNumber(p, 0))
}
