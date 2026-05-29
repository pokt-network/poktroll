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

// BenchmarkSettlementCandidateScan measures the cost of the per-epoch candidate
// sessionEndHeight scan added to the settlement Phase 1 loop (closes the O2
// cross-session window-offset orphan class). Compares:
//   - K=1: legacy parity — no recent params history within the lookback, candidate set
//     collapses to a single E (the same one the legacy single-iterator path computed).
//   - K=2..3: recent window-offset changes — candidate set grows linearly with the number
//     of distinct epochs in the lookback window.
//
// Each sub-bench distributes totalClaims evenly across the K candidate
// sessionEndHeights, then times "compute candidate set + iterate each E's claims
// + collect" on every iteration — exactly the Phase 1 hot path, no settlement
// state mutation so b.N is safe to grow arbitrarily large.
func BenchmarkSettlementCandidateScan(b *testing.B) {
	for _, K := range []int{1, 2, 3} {
		for _, M := range []int{200, 1000, 2551} {
			b.Run(fmt.Sprintf("K=%d/M=%d", K, M), func(b *testing.B) {
				benchSettlementCandidateScan(b, K, M)
			})
		}
	}
}

func benchSettlementCandidateScan(b *testing.B, K, totalClaims int) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(b, nil,
		testkeeper.WithProofRequirement(false),
		testkeeper.WithDefaultModuleBalances(),
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Base shared params with N=4 grid anchored at 1, offsets that pass ValidateBasic
	// for every proofCloseValue candidate below.
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
	require.True(b, ok, "expected a concrete shared keeper")

	// blockHeight is far enough from genesis that the lookback window (4*N = 16 blocks)
	// fits before it and excludes any genesis-seeded entry at height 1.
	const blockHeight int64 = 10_000
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)

	// Distinct ProofWindowCloseOffsetBlocks values per candidate epoch produce distinct
	// sessionEndHeight candidates. Candidate 0 is LIVE; candidates 1..K-1 are recorded
	// in params history at effective_heights inside the lookback window.
	proofCloseValues := []uint64{4, 6, 8}
	require.LessOrEqual(b, K, len(proofCloseValues), "extend proofCloseValues to support K>%d", len(proofCloseValues))

	sessionEndHeights := make([]int64, 0, K)
	for k := 0; k < K; k++ {
		p := sharedParams
		p.ProofWindowCloseOffsetBlocks = proofCloseValues[k]
		tail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&p)
		E := blockHeight - tail - 1
		sessionEndHeights = append(sessionEndHeights, E)

		if k == 0 {
			// Candidate 0 is the live snapshot. No history entry inside the lookback
			// (otherwise it would be picked up + deduped redundantly).
			require.NoError(b, keepers.SharedKeeper.SetParams(sdkCtx, p))
			continue
		}

		// Candidates 1..K-1 live in params history within the lookback window.
		effective := blockHeight - int64(k)
		require.NoError(b, concreteShared.SetParamsAtHeight(sdkCtx, effective, p))
	}

	// Distribute totalClaims evenly across the K candidate sessionEndHeights.
	perE := totalClaims / K
	require.Greater(b, perE, 0, "totalClaims=%d / K=%d must yield at least 1 claim per E", totalClaims, K)
	for ei, E := range sessionEndHeights {
		for m := 0; m < perE; m++ {
			claim := prooftypes.Claim{
				SupplierOperatorAddress: sample.AccAddressBech32(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionId:               fmt.Sprintf("bench-K%d-E%d-m%d", ei, E, m),
					ApplicationAddress:      sample.AccAddressBech32(),
					ServiceId:               "svc1",
					SessionStartBlockHeight: E - 3,
					SessionEndBlockHeight:   E,
				},
			}
			keepers.UpsertClaim(sdkCtx, claim)
		}
	}

	// Sanity check: live candidate set has exactly K distinct Es before the hot loop.
	candidates := keepers.GetExpiringClaimsSessionEndHeights(sdkCtx, blockHeight)
	require.Equal(b, K, len(candidates),
		"setup: expected %d candidate sessionEndHeights, got %d (candidates=%v)",
		K, len(candidates), candidates)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cands := keepers.GetExpiringClaimsSessionEndHeights(sdkCtx, blockHeight)
		for _, E := range cands {
			iter := keepers.GetSessionEndHeightClaimsIterator(sdkCtx, E)
			for ; iter.Valid(); iter.Next() {
				if _, err := iter.Value(); err != nil {
					b.Fatalf("unexpected claim iter error: %v", err)
				}
			}
			iter.Close()
		}
	}
}
