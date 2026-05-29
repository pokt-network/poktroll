//go:build test

package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TestEndBlockerUnbondSuppliers_LongOverdueUnbonding simulates the betanet scenario
// where a supplier unstaked long ago (e.g., at height 23,700) but the EndBlocker
// is called much later (e.g., at height 153,600 — over 120K blocks past unbonding).
// This verifies the EndBlocker can still find and unbond overdue suppliers.
func TestEndBlockerUnbondSuppliers_LongOverdueUnbonding(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	unbondingPeriodSessions := int64(sharedParams.GetSupplierUnbondingPeriodSessions())
	unbondingPeriodBlocks := unbondingPeriodSessions * numBlocksPerSession

	// Stake the supplier
	supplierOperatorAddr := sample.AccAddressBech32()
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(supplierOperatorAddr, supplierOperatorAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify supplier exists and is NOT unbonding
	foundSupplier, isFound := supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, isFound)
	require.False(t, foundSupplier.IsUnbonding())

	// Unstake the supplier
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierOperatorAddr,
		OperatorAddress: supplierOperatorAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	// Verify the supplier is now unbonding
	foundSupplier, isFound = supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierOperatorAddr)
	require.True(t, isFound)
	require.True(t, foundSupplier.IsUnbonding())

	unstakeSessionEndHeight := int64(foundSupplier.GetUnstakeSessionEndHeight())
	unbondingEndHeight := unstakeSessionEndHeight + unbondingPeriodBlocks

	// Activate services at next session start (required for proper state)
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	_, err = supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)

	// --- KEY TEST: Jump WAY past the unbonding end height ---
	// Simulating betanet: unstaked at ~session 395 (height ~23,700),
	// now at session ~2,560 (height ~153,600)
	// That's ~120K blocks past unbonding end.
	longOverdueHeight := unbondingEndHeight + (2000 * numBlocksPerSession) // 2000 sessions past
	// Ensure this lands on a session end height (EndBlocker only runs at session ends)
	longOverdueSessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, longOverdueHeight)
	ctx = keepertest.SetBlockHeight(ctx, longOverdueSessionEnd)

	t.Logf("unstakeSessionEndHeight: %d", unstakeSessionEndHeight)
	t.Logf("unbondingEndHeight: %d", unbondingEndHeight)
	t.Logf("longOverdueSessionEnd: %d (overdue by %d blocks)", longOverdueSessionEnd, longOverdueSessionEnd-unbondingEndHeight)

	// Run the EndBlocker — the supplier should be unbonded even though
	// we're way past the unbonding height
	numUnbonded, err := supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numUnbonded, "EndBlocker should unbond the overdue supplier")

	// Verify the supplier was removed from state
	_, isFound = supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.False(t, isFound, "supplier should be removed from state after unbonding")
}

// TestEndBlockerUnbondSuppliers_MultipleOverdueAtDifferentHeights simulates
// multiple suppliers that unstaked at different heights, all overdue,
// and verifies they all get unbonded in a single EndBlocker call.
func TestEndBlockerUnbondSuppliers_MultipleOverdueAtDifferentHeights(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	unbondingPeriodSessions := int64(sharedParams.GetSupplierUnbondingPeriodSessions())
	unbondingPeriodBlocks := unbondingPeriodSessions * numBlocksPerSession

	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	numSuppliers := 5
	supplierAddrs := make([]string, numSuppliers)

	// Stake all suppliers
	for i := range numSuppliers {
		supplierAddrs[i] = sample.AccAddressBech32()
		stakeMsg, _ := newSupplierStakeMsg(supplierAddrs[i], supplierAddrs[i], initialStake, serviceID)
		_, err := srv.StakeSupplier(ctx, stakeMsg)
		require.NoError(t, err)
	}

	// Unstake suppliers at different session heights, simulating
	// betanet where suppliers unstaked across many different sessions
	var latestUnbondingEnd int64
	for i := range numSuppliers {
		// Advance to a new session for each unstake
		currentHeight := cosmostypes.UnwrapSDKContext(ctx).BlockHeight()
		nextSessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight+numBlocksPerSession*int64(i+1)*10)
		ctx = keepertest.SetBlockHeight(ctx, nextSessionEnd)

		// Activate services at session start before this session end
		sessionStart := sharedtypes.GetSessionStartHeight(&sharedParams, nextSessionEnd)
		tmpCtx := keepertest.SetBlockHeight(ctx, sessionStart)
		_, err := supplierModuleKeepers.BeginBlockerActivateSupplierServices(tmpCtx)
		require.NoError(t, err)

		ctx = keepertest.SetBlockHeight(ctx, nextSessionEnd)

		unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
			Signer:          supplierAddrs[i],
			OperatorAddress: supplierAddrs[i],
		}
		_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
		require.NoError(t, err)

		supplier, _ := supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierAddrs[i])
		require.True(t, supplier.IsUnbonding())

		unbondEnd := int64(supplier.GetUnstakeSessionEndHeight()) + unbondingPeriodBlocks
		if unbondEnd > latestUnbondingEnd {
			latestUnbondingEnd = unbondEnd
		}

		t.Logf("Supplier %d unstaked at session end %d, unbonds at %d",
			i, supplier.GetUnstakeSessionEndHeight(), unbondEnd)
	}

	// Jump way past all unbonding ends
	overdueHeight := latestUnbondingEnd + (500 * numBlocksPerSession)
	overdueSessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, overdueHeight)
	ctx = keepertest.SetBlockHeight(ctx, overdueSessionEnd)

	t.Logf("Running EndBlocker at height %d (latest unbonding end was %d, overdue by %d blocks)",
		overdueSessionEnd, latestUnbondingEnd, overdueSessionEnd-latestUnbondingEnd)

	// Run EndBlocker — ALL suppliers should be unbonded
	numUnbonded, err := supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(numSuppliers), numUnbonded,
		"EndBlocker should unbond all %d overdue suppliers", numSuppliers)

	// Verify none remain
	for i, addr := range supplierAddrs {
		_, isFound := supplierModuleKeepers.GetSupplier(ctx, addr)
		require.False(t, isFound, "supplier %d should be removed from state", i)
	}
}

// TestEndBlockerUnbondSuppliers_OnlyRunsAtSessionEnd verifies that the
// EndBlocker skips unbonding when NOT at a session end height.
func TestEndBlockerUnbondSuppliers_OnlyRunsAtSessionEnd(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	unbondingPeriodSessions := int64(sharedParams.GetSupplierUnbondingPeriodSessions())
	unbondingPeriodBlocks := unbondingPeriodSessions * numBlocksPerSession

	// Stake and unstake a supplier
	supplierAddr := sample.AccAddressBech32()
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(supplierAddr, supplierAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierAddr,
		OperatorAddress: supplierAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	supplier, _ := supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierAddr)
	unbondingEndHeight := int64(supplier.GetUnstakeSessionEndHeight()) + unbondingPeriodBlocks

	// Set block height to past unbonding but NOT at session end
	// Session end heights are multiples of numBlocksPerSession + 1
	pastUnbondingMidSession := unbondingEndHeight + numBlocksPerSession/2
	// Make sure it's NOT a session end
	require.False(t, sharedtypes.IsSessionEndHeight(&sharedParams, pastUnbondingMidSession),
		"test setup: height should not be session end")

	ctx = keepertest.SetBlockHeight(ctx, pastUnbondingMidSession)

	// EndBlocker should skip — returns 0 unbonded
	numUnbonded, err := supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), numUnbonded, "EndBlocker should not run mid-session")

	// Supplier should still exist
	_, isFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isFound, "supplier should still exist since EndBlocker didn't run")

	// Now move to the next session end — it should unbond
	nextSessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, pastUnbondingMidSession)
	ctx = keepertest.SetBlockHeight(ctx, nextSessionEnd)

	numUnbonded, err = supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numUnbonded)
}

// TestEndBlockerUnbondSuppliers_StakeAmountReturned verifies that the
// supplier's staked amount is returned to their account upon unbonding.
func TestEndBlockerUnbondSuppliers_StakeAmountReturned(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	unbondingPeriodSessions := int64(sharedParams.GetSupplierUnbondingPeriodSessions())
	unbondingPeriodBlocks := unbondingPeriodSessions * numBlocksPerSession

	// Stake supplier with a specific amount
	supplierAddr := sample.AccAddressBech32()
	stakeAmount := int64(60_010_000_000) // ~60,010 POKT like betanet suppliers
	stakeMsg, _ := newSupplierStakeMsg(supplierAddr, supplierAddr, stakeAmount, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Check initial balance (negative = deducted from account)
	stakingFee := supplierModuleKeepers.Keeper.GetParams(ctx).StakingFee
	expectedDeduction := -(stakeAmount + stakingFee.Amount.Int64())
	require.Equal(t, expectedDeduction, supplierModuleKeepers.SupplierBalanceMap[supplierAddr])

	// Unstake
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierAddr,
		OperatorAddress: supplierAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	supplier, _ := supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierAddr)
	unbondingEndHeight := int64(supplier.GetUnstakeSessionEndHeight()) + unbondingPeriodBlocks

	// Activate services
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	_, err = supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)

	// Jump far past unbonding (betanet scenario)
	overdueSessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, unbondingEndHeight+(1000*numBlocksPerSession))
	ctx = keepertest.SetBlockHeight(ctx, overdueSessionEnd)

	// Balance should still be deducted (stake locked during unbonding)
	require.Equal(t, expectedDeduction, supplierModuleKeepers.SupplierBalanceMap[supplierAddr],
		"stake should still be locked before EndBlocker runs")

	// Run EndBlocker
	numUnbonded, err := supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numUnbonded)

	// Stake should be returned (only staking fee remains deducted)
	expectedBalanceAfterUnbond := -stakingFee.Amount.Int64()
	require.Equal(t, expectedBalanceAfterUnbond, supplierModuleKeepers.SupplierBalanceMap[supplierAddr],
		"stake should be returned to supplier after unbonding, only staking fee deducted")

	// Verify supplier is gone
	_, isFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.False(t, isFound)
}

// TestEndBlockerUnbondSuppliers_CannotReUnstakeOverdueSupplier verifies that
// a supplier stuck in unbonding state (like betanet's 15 suppliers) cannot
// send another MsgUnstakeSupplier to retry unbonding — the check at line 71
// of msg_server_unstake_supplier.go rejects it with ErrSupplierIsUnstaking,
// even when the supplier is way past its unbonding end height.
func TestEndBlockerUnbondSuppliers_CannotReUnstakeOverdueSupplier(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	unbondingPeriodSessions := int64(sharedParams.GetSupplierUnbondingPeriodSessions())
	unbondingPeriodBlocks := unbondingPeriodSessions * numBlocksPerSession

	// Stake and unstake supplier
	supplierAddr := sample.AccAddressBech32()
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(supplierAddr, supplierAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierAddr,
		OperatorAddress: supplierAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	supplier, _ := supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierAddr)
	require.True(t, supplier.IsUnbonding())

	unbondingEndHeight := int64(supplier.GetUnstakeSessionEndHeight()) + unbondingPeriodBlocks

	// Jump WAY past unbonding end — simulating betanet's 120K+ blocks overdue
	overdueHeight := unbondingEndHeight + (2000 * numBlocksPerSession)
	ctx = keepertest.SetBlockHeight(ctx, overdueHeight)

	// Try to unstake again — should FAIL even though we're way past unbonding end
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierIsUnstaking.Error())

	t.Logf("Confirmed: cannot re-unstake supplier %d blocks past unbonding end", overdueHeight-unbondingEndHeight)

	// But re-staking SHOULD work — it clears UnstakeSessionEndHeight
	restakeMsg, _ := newSupplierStakeMsg(supplierAddr, supplierAddr, initialStake+1, serviceID)
	_, err = srv.StakeSupplier(ctx, restakeMsg)
	require.NoError(t, err)

	// Verify supplier is no longer unbonding
	supplier, isFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, isFound)
	require.False(t, supplier.IsUnbonding(), "re-staking should clear UnstakeSessionEndHeight")

	t.Log("Confirmed: re-staking clears unbonding state — this is the only recovery path")
}

// TestEndBlockerUnbondSuppliers_IndexIntegrity verifies that the unstaking
// index is properly maintained — specifically that a supplier appears in
// the iterator after unstaking and is removed after unbonding.
func TestEndBlockerUnbondSuppliers_IndexIntegrity(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	unbondingPeriodSessions := int64(sharedParams.GetSupplierUnbondingPeriodSessions())
	unbondingPeriodBlocks := unbondingPeriodSessions * numBlocksPerSession

	// Stake supplier
	supplierAddr := sample.AccAddressBech32()
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(supplierAddr, supplierAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Before unstaking: iterator should be empty
	iter := supplierModuleKeepers.GetAllUnstakingSuppliersIterator(ctx)
	require.False(t, iter.Valid(), "no suppliers should be in unstaking index before unstake")
	iter.Close()

	// Unstake supplier
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierAddr,
		OperatorAddress: supplierAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	// After unstaking: iterator should contain this supplier
	iter = supplierModuleKeepers.GetAllUnstakingSuppliersIterator(ctx)
	require.True(t, iter.Valid(), "supplier should appear in unstaking index after unstake")
	indexedAddr := string(iter.Value())
	require.Equal(t, supplierAddr, indexedAddr)
	iter.Close()

	supplier, _ := supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierAddr)
	unbondingEndHeight := int64(supplier.GetUnstakeSessionEndHeight()) + unbondingPeriodBlocks

	// Activate services
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	_, err = supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)

	// Move to unbonding end and run EndBlocker
	sessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, unbondingEndHeight)
	ctx = keepertest.SetBlockHeight(ctx, sessionEnd)

	numUnbonded, err := supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numUnbonded)

	// After unbonding: iterator should be empty again
	iter = supplierModuleKeepers.GetAllUnstakingSuppliersIterator(ctx)
	require.False(t, iter.Valid(), "unstaking index should be empty after unbonding")
	iter.Close()
}

// TestEndBlockerUnbondSuppliers_NumBlocksPerSessionDecreaseDoesNotReleaseEarly
// is the encoded F1 (#543) regression test for the supplier side.
//
// A supplier that began unbonding under N=oldN must NOT be released early when
// num_blocks_per_session is decreased to a smaller value mid-unbonding. Releasing
// early would shorten the unbonding window below what was promised at unstake
// time, potentially letting the supplier withdraw its stake before in-flight
// claims settle. F1 ("funds-loss on N decrease") was identified as a blocking
// precondition for the anchored session grid and is fixed in EndBlockerUnbondSuppliers
// by computing the unbonding end height via GetParamsAtHeight(unstakeSessionEndHeight)
// — the shared params epoch effective at the time the supplier began unbonding —
// instead of live params.
//
// Sequence exercised end-to-end:
//   1. N=oldN; stake + unstake supplier → captures unstakeSessionEndHeight.
//   2. Plant a NEW shared-params epoch with N=newN (smaller) at the boundary
//      AFTER unstakeSessionEndHeight, simulating governance promotion of a
//      DEFERRED num_blocks_per_session change.
//   3. Compute the "buggy" early unbonding end the EndBlocker would land on if
//      it read LIVE params at runtime (unstakeSessionEnd + unbonding_sessions * newN).
//   4. Walk to that early height (at a session boundary under newN) and call
//      EndBlockerUnbondSuppliers. Under the bug the supplier is removed here;
//      under the fix it stays (because GetParamsAtHeight returns oldN's epoch
//      at the unstake height).
//   5. Walk to the TRUE oldN-derived unbonding horizon. EndBlocker releases the
//      supplier here, confirming the original commitment was honored.
func TestEndBlockerUnbondSuppliers_NumBlocksPerSessionDecreaseDoesNotReleaseEarly(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// --- Set N=oldN at the genesis grid -----------------------------------------
	// Cast SharedKeeper to its concrete type so we can plant a historical params
	// epoch via SetParamsAtHeight — the same shape governance + the shared
	// EndBlocker produce when promoting a deferred num_blocks_per_session change.
	concreteShared, castOK := supplierModuleKeepers.SharedKeeper.(sharedkeeper.Keeper)
	require.True(t, castOK, "test setup: expected concrete shared keeper")

	const (
		oldN                   int64 = 20
		newN                   int64 = 4 // halving direction; F1 protects the unbonding actor
		unbondingPeriodSessions int64 = 8
	)

	sharedParams := concreteShared.GetParams(ctx)
	sharedParams.NumBlocksPerSession = uint64(oldN)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.SupplierUnbondingPeriodSessions = uint64(unbondingPeriodSessions)
	sharedParams.ApplicationUnbondingPeriodSessions = uint64(unbondingPeriodSessions)
	sharedParams.GatewayUnbondingPeriodSessions = uint64(unbondingPeriodSessions)
	require.NoError(t, concreteShared.SetParams(ctx, sharedParams))
	// Also seed the OLD epoch in params history at effective_height=1 — mirrors the
	// v0.1.34 upgrade handler's grid seed. Without this, GetParamsAtHeight(unstakeHeight)
	// would fall through to live params (which we mutate below) and the test would not
	// exercise the historical-epoch resolution path.
	require.NoError(t, concreteShared.SetParamsAtHeight(ctx, 1, sharedParams))

	// --- Stake supplier ---------------------------------------------------------
	supplierAddr := sample.AccAddressBech32()
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(supplierAddr, supplierAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// --- Unstake at a session-end height under N=oldN ---------------------------
	ctx = setBlockHeightToNextSessionStart(ctx, supplierModuleKeepers.SharedKeeper)
	_, err = supplierModuleKeepers.BeginBlockerActivateSupplierServices(ctx)
	require.NoError(t, err)

	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierAddr,
		OperatorAddress: supplierAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	unbondingSupplier, found := supplierModuleKeepers.GetDehydratedSupplier(ctx, supplierAddr)
	require.True(t, found)
	require.True(t, unbondingSupplier.IsUnbonding())

	unstakeSessionEndHeight := int64(unbondingSupplier.GetUnstakeSessionEndHeight())

	// Two horizons whose ordering is the whole point of the test:
	//   - oldN: what the supplier was promised at unstake time
	//   - newN: where the bug would release it early
	unbondingEndUnderOldN := unstakeSessionEndHeight + unbondingPeriodSessions*oldN
	unbondingEndUnderNewN := unstakeSessionEndHeight + unbondingPeriodSessions*newN

	require.Less(t, unbondingEndUnderNewN, unbondingEndUnderOldN,
		"test setup: newN must produce an EARLIER unbonding end than oldN — otherwise the test cannot exercise F1")

	// --- Plant a new shared-params epoch with N=newN at the next session boundary
	// AFTER the unstake. This simulates the shape governance + the deferred
	// promotion produce: a fresh params-history entry whose effective_height is
	// a session boundary, leaving the unstake height in the OLDER epoch.
	postUnstakeBoundary := unstakeSessionEndHeight + 1
	newEpochParams := sharedParams
	newEpochParams.NumBlocksPerSession = uint64(newN)
	require.NoError(t, concreteShared.SetParamsAtHeight(ctx, postUnstakeBoundary, newEpochParams))
	// Live params should now reflect the new epoch — write them through too so
	// IsSessionEndHeight queries from the EndBlocker see N=newN at runtime.
	require.NoError(t, concreteShared.SetParams(ctx, newEpochParams))

	// Sanity: GetParamsAtHeight at the unstake height must still return oldN —
	// otherwise the test is no longer exercising the cross-epoch resolution.
	paramsAtUnstake := concreteShared.GetParamsAtHeight(ctx, unstakeSessionEndHeight)
	require.Equal(t, uint64(oldN), paramsAtUnstake.NumBlocksPerSession,
		"GetParamsAtHeight at unstake height must return the OLD epoch — otherwise we're not testing F1")

	// --- The critical assertion: do NOT release at the newN-derived horizon ---
	// Walk to the first session-end height at or after unbondingEndUnderNewN
	// under live (newN) params.
	earlyHorizon := sharedtypes.GetSessionEndHeight(&newEpochParams, unbondingEndUnderNewN)
	ctx = keepertest.SetBlockHeight(ctx, earlyHorizon)

	require.Less(t, earlyHorizon, unbondingEndUnderOldN,
		"test setup: earlyHorizon must be strictly before the real oldN-derived horizon")

	numUnbonded, err := supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), numUnbonded,
		"F1 VIOLATION: supplier released at newN-derived early horizon (%d) — must wait until the real oldN-derived horizon (%d)",
		earlyHorizon, unbondingEndUnderOldN)

	_, stillFound := supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.True(t, stillFound,
		"F1 VIOLATION: supplier removed from state before oldN-derived unbonding horizon")

	// --- Sanity: at the true (oldN-derived) horizon the supplier IS released ---
	// Walk to the first session-end height at or after unbondingEndUnderOldN.
	finalHorizon := sharedtypes.GetSessionEndHeight(&newEpochParams, unbondingEndUnderOldN)
	if finalHorizon < unbondingEndUnderOldN {
		// In edge cases where unbondingEndUnderOldN itself lands mid-session
		// under live N=newN, jump one full newN-session forward to ensure we
		// cross the threshold at a real session-end.
		finalHorizon += newN
	}
	ctx = keepertest.SetBlockHeight(ctx, finalHorizon)

	numUnbonded, err = supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numUnbonded,
		"supplier should unbond at or after its real oldN-derived horizon (%d) — got 0 at height %d",
		unbondingEndUnderOldN, finalHorizon)

	_, stillFound = supplierModuleKeepers.GetSupplier(ctx, supplierAddr)
	require.False(t, stillFound,
		"supplier should be removed from state at its real oldN-derived unbonding horizon")
}
