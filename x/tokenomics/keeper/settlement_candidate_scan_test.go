package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestSettlementCandidateScan_MultiEpochCorrectness exercises the per-epoch
// candidate sessionEndHeight scan added by the O2 fix at K=3 (live + two recent
// params-history epochs, each with distinct ProofWindowCloseOffsetBlocks).
//
// For each candidate epoch the test inserts a real Claim record at the
// sessionEndHeight that epoch's offsets would resolve to at the chosen
// blockHeight, then asserts:
//   - GetExpiringClaimsSessionEndHeights returns exactly K=3 distinct candidates.
//   - The candidate set is exactly the expected sessionEndHeight values.
//   - Iterating each candidate's claim store prefix yields its corresponding claim.
//
// Without the candidate scan (legacy single-iterator path), only the
// live-derived candidate would be iterated and the two history-derived claims
// would be missed — the cross-session window-offset orphan class (O2).
func TestSettlementCandidateScan_MultiEpochCorrectness(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil,
		testkeeper.WithProofRequirement(false),
		testkeeper.WithDefaultModuleBalances(),
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Base shared params with N=4 grid anchored at 1. Each of the three epochs
	// below differs ONLY in ProofWindowCloseOffsetBlocks so that
	// GetSessionEndToProofWindowCloseBlocks (and therefore the derived candidate
	// sessionEndHeight) is distinct per epoch.
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = 4
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.GracePeriodEndOffsetBlocks = 1
	sharedParams.ClaimWindowOpenOffsetBlocks = 1
	sharedParams.ClaimWindowCloseOffsetBlocks = 2
	sharedParams.ProofWindowOpenOffsetBlocks = 0
	sharedParams.SupplierUnbondingPeriodSessions = 16
	sharedParams.ApplicationUnbondingPeriodSessions = 16
	sharedParams.GatewayUnbondingPeriodSessions = 16

	concreteShared, ok := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected a concrete shared keeper")

	// blockHeight is far enough from genesis that the lookback window (4*N = 16
	// blocks) fits before it; far enough that no genesis-seeded entry at height 1
	// falls inside the lookback.
	const blockHeight int64 = 10_000
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)

	// Three distinct ProofWindowCloseOffsetBlocks values produce three distinct
	// candidate sessionEndHeights at blockHeight. Index 0 is the LIVE epoch
	// (current shared params); indices 1 and 2 sit in params history within the
	// lookback window.
	proofCloseValues := [3]uint64{4, 6, 8}
	expectedSessionEndHeights := make([]int64, 0, 3)

	for k, proofClose := range proofCloseValues {
		epochParams := sharedParams
		epochParams.ProofWindowCloseOffsetBlocks = proofClose
		tail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&epochParams)
		E := blockHeight - tail - 1
		expectedSessionEndHeights = append(expectedSessionEndHeights, E)

		if k == 0 {
			// Live epoch — no history entry inside the lookback (otherwise the
			// scan would pick it up redundantly and dedupe).
			require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, epochParams))
			continue
		}

		// History epochs at distinct effective_heights within the lookback. Spacing
		// at -k blocks keeps every entry in the window and ordered.
		effective := blockHeight - int64(k)
		require.NoError(t, concreteShared.SetParamsAtHeight(sdkCtx, effective, epochParams))
	}

	// Insert one Claim at each expected sessionEndHeight, tagged so we can verify
	// the right claim was found under the right candidate.
	expectedClaimBySessionEnd := make(map[int64]string, 3)
	for k, E := range expectedSessionEndHeights {
		supplierAddr := sample.AccAddressBech32()
		sessionID := fmt.Sprintf("epoch-%d-E%d", k, E)
		expectedClaimBySessionEnd[E] = sessionID

		claim := prooftypes.Claim{
			SupplierOperatorAddress: supplierAddr,
			SessionHeader: &sessiontypes.SessionHeader{
				SessionId:               sessionID,
				ApplicationAddress:      sample.AccAddressBech32(),
				ServiceId:               "svc1",
				SessionStartBlockHeight: E - 3,
				SessionEndBlockHeight:   E,
			},
		}
		keepers.UpsertClaim(sdkCtx, claim)
	}

	// The per-epoch candidate scan must return exactly K=3 distinct candidates,
	// matching the expected set.
	candidates := keepers.GetExpiringClaimsSessionEndHeights(sdkCtx, blockHeight)
	require.Len(t, candidates, 3,
		"expected exactly 3 candidate sessionEndHeights (one per epoch), got %d: %v",
		len(candidates), candidates)
	require.ElementsMatch(t, expectedSessionEndHeights, candidates,
		"candidate sessionEndHeights must match the per-epoch derived set")

	// Iterating each candidate's claim store prefix yields the corresponding claim.
	foundSessionIDs := make(map[int64]string, 3)
	for _, E := range candidates {
		iter := keepers.GetSessionEndHeightClaimsIterator(sdkCtx, E)
		count := 0
		for ; iter.Valid(); iter.Next() {
			claim, iterErr := iter.Value()
			require.NoError(t, iterErr, "unexpected claim iter error at sessionEndHeight=%d", E)
			foundSessionIDs[E] = claim.GetSessionHeader().GetSessionId()
			count++
		}
		iter.Close()
		require.Equal(t, 1, count,
			"each candidate sessionEndHeight should contain exactly one claim; sessionEndHeight=%d has %d",
			E, count)
	}
	require.Equal(t, expectedClaimBySessionEnd, foundSessionIDs,
		"each expected claim must be located at its corresponding candidate sessionEndHeight")
}
