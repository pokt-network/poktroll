package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestEndBlockerPruneSupplierServiceConfigHistory_RespectsProofWindow(t *testing.T) {
	// This test verifies the core fix: service configs are kept until AFTER
	// the proof window closes for the LAST ACTIVE SESSION, not the session
	// at the deactivation height.

	t.Run("config kept when proof window for last active session hasn't closed", func(t *testing.T) {
		// Create fresh keepers for this test
		keepers, ctx := keepertest.SupplierKeeper(t)
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		supplierKeeper := keepers.Keeper
		sharedParams := keepers.SharedKeeper.GetParams(ctx)

		serviceId := "test-service"
		numBlocksPerSession := int64(sharedParams.NumBlocksPerSession)
		supplierOperatorAddr := sample.AccAddressBech32()

		// Calculate session boundaries
		// Session 1: blocks 1 to numBlocksPerSession
		// Session 2: blocks numBlocksPerSession+1 to 2*numBlocksPerSession
		sessionOneEnd := numBlocksPerSession
		sessionTwoStart := sessionOneEnd + 1

		// Deactivation height is set to first block of next session (simulating unstake)
		deactivationHeight := sessionTwoStart

		// Last active block is the last block of session 1
		lastActiveHeight := deactivationHeight - 1 // = sessionOneEnd

		// Calculate when the proof window closes for the last active session
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, lastActiveHeight)

		// Set current height to just before the proof window closes
		currentHeight := proofWindowCloseHeight - 1
		sdkCtx = sdkCtx.WithBlockHeight(currentHeight)
		testCtx := sdkCtx

		services := []*sharedtypes.SupplierServiceConfig{{ServiceId: serviceId}}
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierOperatorAddr,
			services,
			1,                  // activated at block 1
			deactivationHeight, // deactivated at first block of session 2
		)

		supplier := sharedtypes.Supplier{
			OperatorAddress:      supplierOperatorAddr,
			Services:             services,
			ServiceConfigHistory: serviceConfigHistory,
		}
		supplierKeeper.SetAndIndexDehydratedSupplier(testCtx, supplier)

		// Run pruning - should NOT prune because proof window for last active session hasn't closed
		numPruned, err := supplierKeeper.EndBlockerPruneSupplierServiceConfigHistory(testCtx)
		require.NoError(t, err)
		require.Equal(t, 0, numPruned, "should not prune - proof window for last active session hasn't closed")

		// Verify config still exists
		rehydratedSupplier, found := supplierKeeper.GetSupplier(testCtx, supplierOperatorAddr)
		require.True(t, found)
		require.Len(t, rehydratedSupplier.ServiceConfigHistory, 1, "config should still exist")
	})

	t.Run("config pruned after proof window for last active session closes", func(t *testing.T) {
		// Create fresh keepers for this test
		keepers, ctx := keepertest.SupplierKeeper(t)
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		supplierKeeper := keepers.Keeper
		sharedParams := keepers.SharedKeeper.GetParams(ctx)

		serviceId := "test-service"
		numBlocksPerSession := int64(sharedParams.NumBlocksPerSession)
		supplierOperatorAddr := sample.AccAddressBech32()

		// Calculate session boundaries
		sessionOneEnd := numBlocksPerSession
		sessionTwoStart := sessionOneEnd + 1

		// Deactivation height is set to first block of next session
		deactivationHeight := sessionTwoStart

		// Last active block is the last block of session 1
		lastActiveHeight := deactivationHeight - 1

		// Calculate when the proof window closes for the last active session
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, lastActiveHeight)

		// Set current height to AFTER the proof window closes
		currentHeight := proofWindowCloseHeight + 1
		sdkCtx = sdkCtx.WithBlockHeight(currentHeight)
		testCtx := sdkCtx

		services := []*sharedtypes.SupplierServiceConfig{{ServiceId: serviceId}}
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierOperatorAddr,
			services,
			1,                  // activated at block 1
			deactivationHeight, // deactivated at first block of session 2
		)

		supplier := sharedtypes.Supplier{
			OperatorAddress:      supplierOperatorAddr,
			Services:             services,
			ServiceConfigHistory: serviceConfigHistory,
		}
		supplierKeeper.SetAndIndexDehydratedSupplier(testCtx, supplier)

		// Run pruning - SHOULD prune because proof window for last active session has closed
		numPruned, err := supplierKeeper.EndBlockerPruneSupplierServiceConfigHistory(testCtx)
		require.NoError(t, err)
		require.Equal(t, 1, numPruned, "should prune - proof window for last active session has closed")

		// Verify config was pruned
		rehydratedSupplier, found := supplierKeeper.GetSupplier(testCtx, supplierOperatorAddr)
		require.True(t, found)
		require.Len(t, rehydratedSupplier.ServiceConfigHistory, 0, "config should be pruned")
	})

	t.Run("uses correct session for proof window calculation (not deactivation session)", func(t *testing.T) {
		// This test verifies the fix: we use DeactivationHeight-1 (last active block)
		// to calculate the proof window, NOT the deactivation height itself.
		//
		// If we incorrectly used DeactivationHeight, we'd calculate the proof window
		// for the NEXT session (where the config isn't even active), resulting in
		// configs being kept longer than necessary.

		// Create fresh keepers for this test
		keepers, ctx := keepertest.SupplierKeeper(t)
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		supplierKeeper := keepers.Keeper
		sharedParams := keepers.SharedKeeper.GetParams(ctx)

		serviceId := "test-service"
		numBlocksPerSession := int64(sharedParams.NumBlocksPerSession)
		supplierOperatorAddr := sample.AccAddressBech32()

		// Calculate session boundaries
		sessionOneEnd := numBlocksPerSession
		sessionTwoStart := sessionOneEnd + 1
		sessionTwoEnd := 2 * numBlocksPerSession

		// Deactivation height is set to first block of session 2
		deactivationHeight := sessionTwoStart

		// Calculate proof windows for both sessions
		// Session 1 (last active): proof window based on lastActiveHeight = deactivationHeight - 1
		lastActiveHeight := deactivationHeight - 1
		proofWindowCloseSession1 := sharedtypes.GetProofWindowCloseHeight(&sharedParams, lastActiveHeight)

		// Session 2 (deactivation session - config NOT active here)
		proofWindowCloseSession2 := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionTwoEnd)

		// Verify session 2's proof window closes AFTER session 1's
		require.Greater(t, proofWindowCloseSession2, proofWindowCloseSession1,
			"session 2 proof window should close after session 1")

		// Set current height between the two proof windows
		// If the fix is correct, config SHOULD be pruned (session 1 window closed)
		// If the fix is wrong (using deactivationHeight), config would NOT be pruned (session 2 window open)
		currentHeight := proofWindowCloseSession1 + 1
		require.Less(t, currentHeight, proofWindowCloseSession2,
			"test setup: current height should be between the two proof windows")

		sdkCtx = sdkCtx.WithBlockHeight(currentHeight)
		testCtx := sdkCtx

		services := []*sharedtypes.SupplierServiceConfig{{ServiceId: serviceId}}
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierOperatorAddr,
			services,
			1,                  // activated at block 1
			deactivationHeight, // deactivated at first block of session 2
		)

		supplier := sharedtypes.Supplier{
			OperatorAddress:      supplierOperatorAddr,
			Services:             services,
			ServiceConfigHistory: serviceConfigHistory,
		}
		supplierKeeper.SetAndIndexDehydratedSupplier(testCtx, supplier)

		// Run pruning - with the correct fix, config SHOULD be pruned
		numPruned, err := supplierKeeper.EndBlockerPruneSupplierServiceConfigHistory(testCtx)
		require.NoError(t, err)
		require.Equal(t, 1, numPruned,
			"config should be pruned because we use lastActiveHeight (session 1), not deactivationHeight (session 2)")
	})
}
