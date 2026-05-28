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

// TestSettlementCandidateScan_GenesisSeededOldEpochOnLongRunningChain is the
// regression test for the O2 lookback-bound bug surfaced in audit pass 3.
//
// Scenario (mainnet v0.1.34 long-running chain + FIRST post-upgrade window-offset change):
//   - v0.1.34 upgrade handler seeds params history at h=1 with the genesis epoch's params.
//     That entry persists for the lifetime of the chain and is the ONLY history representation
//     of the "OLD" epoch immediately after the first post-upgrade window-offset change.
//   - At some later block (>>240 blocks past h=1), governance changes window offsets via
//     `MsgUpdateParam`, which writes a NEW history entry at the next session boundary.
//   - A claim created in the LAST session under the OLD epoch must still be locatable at
//     settlement time, even though its stored sessionEndHeight resolves only under h=1's
//     offsets — not under the (now-live) NEW offsets.
//
// Pre-fix bug:
//
//	The legacy `candidateSessionEndHeightsForLiveParams` had a `max(4*N, 240)` lookback
//	bound. On a long-running chain, h=1 falls far below `blockHeight - 240`, so the
//	reverse iterator's `effHeight < earliest` check fires immediately on h=1 → STOP →
//	OLD epoch never produces a candidate → claim orphaned forever.
//
// This test pins the fix that drops the `earliest` global stop and replaces it with a
// per-epoch in-flight check that lets the iterator continue past the genesis-seeded
// entry.
func TestSettlementCandidateScan_GenesisSeededOldEpochOnLongRunningChain(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil,
		testkeeper.WithProofRequirement(false),
		testkeeper.WithDefaultModuleBalances(),
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Choose params so the OLD epoch's LAST possible claim has its proof-window-close
	// EXACTLY at blockHeight. That maximises the in-flight overlap: at blockHeight,
	// the OLD epoch still owns one settling claim. Pre-fix bug would have skipped
	// h=1 because its effHeight < (blockHeight - max(4*N, 240)) → orphaned.
	const (
		N           int64 = 4
		blockHeight int64 = 720_301
		// OLD epoch's last session ends at newEpochEff - 1 = 720_240. Its proof window
		// closes at 720_240 + oldTail + 1. Choose oldTail = 60 so close == 720_301.
		oldTail     uint64 = 60
		newEpochEff int64  = 720_241
	)

	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = uint64(N)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.GracePeriodEndOffsetBlocks = 1
	sharedParams.SupplierUnbondingPeriodSessions = 16
	sharedParams.ApplicationUnbondingPeriodSessions = 16
	sharedParams.GatewayUnbondingPeriodSessions = 16

	// OLD epoch: long proof-window-close offset so a claim ending at 720_240 still
	// has its proof window open at 720_301.
	oldEpochParams := sharedParams
	oldEpochParams.ClaimWindowOpenOffsetBlocks = 0
	oldEpochParams.ClaimWindowCloseOffsetBlocks = 0
	oldEpochParams.ProofWindowOpenOffsetBlocks = 0
	oldEpochParams.ProofWindowCloseOffsetBlocks = oldTail

	// NEW epoch (LIVE at blockHeight): different (smaller) offsets so the candidate
	// sessionEndHeight derived from live params is DIFFERENT from the OLD one.
	newEpochParams := sharedParams
	newEpochParams.ClaimWindowOpenOffsetBlocks = 0
	newEpochParams.ClaimWindowCloseOffsetBlocks = 0
	newEpochParams.ProofWindowOpenOffsetBlocks = 0
	newEpochParams.ProofWindowCloseOffsetBlocks = 30

	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, newEpochParams))

	concreteShared, ok := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected a concrete shared keeper")

	// History h=1 = OLD epoch (the v0.1.34 upgrade-handler-seeded entry — note
	// blockHeight - 1 == 720_300, FAR above any reasonable max(4*N, 240) floor).
	require.NoError(t, concreteShared.SetParamsAtHeight(sdkCtx, 1, oldEpochParams))
	// History h=newEpochEff = NEW epoch (mirrors what `MsgUpdateParam` would write).
	require.NoError(t, concreteShared.SetParamsAtHeight(sdkCtx, newEpochEff, newEpochParams))

	// Sanity-derive the two candidate sessionEndHeights at the given blockHeight under
	// each epoch's offsets. These MUST both appear in the result; the OLD one is what
	// the pre-fix bug dropped.
	oldComputedTail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&oldEpochParams)
	newComputedTail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&newEpochParams)
	expectedOldCandidate := blockHeight - int64(oldComputedTail) - 1
	expectedNewCandidate := blockHeight - int64(newComputedTail) - 1
	require.NotEqual(t, expectedOldCandidate, expectedNewCandidate,
		"test setup invariant: OLD and NEW candidate sessionEndHeights must differ")

	// Insert a claim at the OLD-epoch-derived sessionEndHeight. This is the claim
	// that the pre-fix bug would orphan.
	oldClaimSessionID := "old-epoch-claim"
	oldClaim := prooftypes.Claim{
		SupplierOperatorAddress: sample.AccAddressBech32(),
		SessionHeader: &sessiontypes.SessionHeader{
			SessionId:               oldClaimSessionID,
			ApplicationAddress:      sample.AccAddressBech32(),
			ServiceId:               "svc1",
			SessionStartBlockHeight: expectedOldCandidate - 3,
			SessionEndBlockHeight:   expectedOldCandidate,
		},
	}
	keepers.UpsertClaim(sdkCtx, oldClaim)

	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)

	candidates := keepers.GetExpiringClaimsSessionEndHeights(sdkCtx, blockHeight)

	require.Contains(t, candidates, expectedOldCandidate,
		"O2 regression: OLD-epoch candidate sessionEndHeight (%d) must be in the candidate set "+
			"even though its history entry (h=1) is far below any `blockHeight - max(4*N, 240)` floor",
		expectedOldCandidate)
	require.Contains(t, candidates, expectedNewCandidate,
		"NEW-epoch candidate (%d) must always be in the set (derived from live params)",
		expectedNewCandidate)

	// Confirm the claim can actually be located via the candidate path.
	iter := keepers.GetSessionEndHeightClaimsIterator(sdkCtx, expectedOldCandidate)
	defer iter.Close()
	require.True(t, iter.Valid(), "iterator over OLD-candidate sessionEndHeight must yield the seeded claim")
	got, getErr := iter.Value()
	require.NoError(t, getErr)
	require.Equal(t, oldClaimSessionID, got.GetSessionHeader().GetSessionId(),
		"claim located via OLD-epoch candidate must match the seeded claim")
}
