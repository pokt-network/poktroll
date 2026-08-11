package keeper_test

import (
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

// TestSettlementRipeness_ShrunkOffsetsDoNotSettleOldEpochClaimsEarly pins the
// ripeness gate in SettlePendingClaims Phase 1.
//
// candidateSessionEndHeights is a deliberate SUPERSET: every params epoch contributes
// the session-end height it would settle at the current block, with no check that the
// epoch actually governs that height. When governance SHRINKS the sum of the claim/proof
// window offsets, the LIVE epoch's candidate reaches back into the previous (longer-tail)
// epoch and names a real session-end height whose claims are not yet ripe -- their proof
// window, resolved at their own session-end height, is still open.
//
// Without the gate those claims settle early. Every proof not yet submitted is recorded
// PROOF_MISSING, the supplier is slashed, and the claim is removed, so a later
// MsgSubmitProof cannot recover it. Nothing downstream re-checks ripeness.
//
// The claim MUST simply be deferred, not dropped: at the block where it is genuinely
// ripe, the epoch owning its session-end height names that height again.
func TestSettlementRipeness_ShrunkOffsetsDoNotSettleOldEpochClaimsEarly(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil,
		testkeeper.WithProofRequirement(false),
		testkeeper.WithDefaultModuleBalances(),
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	concreteShared, ok := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected a concrete shared keeper")

	// Grid shared by both epochs; only ProofWindowCloseOffsetBlocks differs, so the two
	// epochs derive different candidate session-end heights at the same block.
	baseParams := keepers.SharedKeeper.GetParams(sdkCtx)
	baseParams.NumBlocksPerSession = 4
	baseParams.SessionGridAnchorHeight = 1
	baseParams.SessionNumberAtAnchor = 1
	baseParams.GracePeriodEndOffsetBlocks = 1
	baseParams.ClaimWindowOpenOffsetBlocks = 1
	baseParams.ClaimWindowCloseOffsetBlocks = 2
	baseParams.ProofWindowOpenOffsetBlocks = 0
	baseParams.SupplierUnbondingPeriodSessions = 16
	baseParams.ApplicationUnbondingPeriodSessions = 16
	baseParams.GatewayUnbondingPeriodSessions = 16

	// OLD epoch: long tail. NEW epoch: governance shrinks the proof window close offset.
	oldParams := baseParams
	oldParams.ProofWindowCloseOffsetBlocks = 8
	newParams := baseParams
	newParams.ProofWindowCloseOffsetBlocks = 2

	oldTail := int64(sharedtypes.GetSessionEndToProofWindowCloseBlocks(&oldParams))
	newTail := int64(sharedtypes.GetSessionEndToProofWindowCloseBlocks(&newParams))
	require.Greater(t, oldTail, newTail, "test setup: the new epoch must have a shorter tail")

	const blockHeight int64 = 10_000
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)

	// The height the LIVE (shrunk) epoch proposes at blockHeight. Because the tail
	// shrank, this reaches back into the OLD epoch's territory.
	prematureSessionEndHeight := blockHeight - newTail - 1

	// Make the OLD epoch govern that height, and the NEW epoch take effect after it.
	require.NoError(t, concreteShared.SetParamsAtHeight(sdkCtx, 1, oldParams))
	require.NoError(t, concreteShared.SetParamsAtHeight(sdkCtx, prematureSessionEndHeight+1, newParams))
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, newParams))

	// Sanity: resolved at its own session-end height, this claim's proof window is
	// still open at blockHeight -- it is genuinely not ripe.
	paramsAtSessionEnd := keepers.SharedKeeper.GetParamsAtHeight(sdkCtx, prematureSessionEndHeight)
	proofWindowCloseHeight := prematureSessionEndHeight +
		int64(sharedtypes.GetSessionEndToProofWindowCloseBlocks(&paramsAtSessionEnd))
	require.GreaterOrEqual(t, proofWindowCloseHeight, blockHeight,
		"test setup: the claim's proof window must still be open at the settlement block")

	claim := prooftypes.Claim{
		SupplierOperatorAddress: sample.AccAddressBech32(),
		SessionHeader: &sessiontypes.SessionHeader{
			SessionId:               "not-yet-ripe",
			ApplicationAddress:      sample.AccAddressBech32(),
			ServiceId:               "svc1",
			SessionStartBlockHeight: prematureSessionEndHeight - 3,
			SessionEndBlockHeight:   prematureSessionEndHeight,
		},
	}
	keepers.UpsertClaim(sdkCtx, claim)

	// The candidate scan still proposes the height -- the superset is intentional.
	candidates := keepers.GetExpiringClaimsSessionEndHeights(sdkCtx, blockHeight)
	require.Contains(t, candidates, prematureSessionEndHeight,
		"the shrunk live epoch is expected to propose this height; the gate must reject the CLAIM, not the candidate")

	settledResults, expiredResults, _, err := keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// The claim must neither settle nor expire: it is deferred.
	require.Equal(t, 0, len(settledResults),
		"a claim whose proof window is still open must not be settled early")
	require.Equal(t, 0, len(expiredResults),
		"a claim whose proof window is still open must not be expired (that would slash the supplier as PROOF_MISSING)")

	// It must still be in the store, recoverable at its real settlement block.
	iter := keepers.GetSessionEndHeightClaimsIterator(sdkCtx, prematureSessionEndHeight)
	remaining := 0
	for ; iter.Valid(); iter.Next() {
		remaining++
	}
	iter.Close()
	require.Equal(t, 1, remaining,
		"the deferred claim must remain in the store; removing it makes a later MsgSubmitProof impossible")
}
