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
	// the proof window closes, not just until the deactivation height.

	keepers, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	supplierKeeper := keepers.Keeper
	sharedParams := keepers.SharedKeeper.GetParams(ctx)

	supplierOperatorAddr := sample.AccAddressBech32()
	serviceId := "test-service"

	t.Run("config kept when proof window hasn't closed", func(t *testing.T) {
		// Create a config deactivated at the current height
		currentHeight := int64(100)
		sdkCtx = sdkCtx.WithBlockHeight(currentHeight)
		testCtx := sdkCtx

		services := []*sharedtypes.SupplierServiceConfig{{ServiceId: serviceId}}
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierOperatorAddr,
			services,
			1,
			currentHeight, // deactivated at current height
		)

		supplier := sharedtypes.Supplier{
			OperatorAddress:      supplierOperatorAddr,
			Services:             services,
			ServiceConfigHistory: serviceConfigHistory,
		}
		supplierKeeper.SetAndIndexDehydratedSupplier(testCtx, supplier)

		// At deactivation height, proof window hasn't closed yet
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, currentHeight)
		require.Greater(t, proofWindowCloseHeight, currentHeight,
			"proof window should close AFTER deactivation height")

		// Run pruning - should NOT prune because proof window hasn't closed
		numPruned, err := supplierKeeper.EndBlockerPruneSupplierServiceConfigHistory(testCtx)
		require.NoError(t, err)
		require.Equal(t, 0, numPruned, "should not prune - proof window hasn't closed")

		// Verify config still exists
		rehydratedSupplier, found := supplierKeeper.GetSupplier(testCtx, supplierOperatorAddr)
		require.True(t, found)
		require.Len(t, rehydratedSupplier.ServiceConfigHistory, 1)
	})
}
