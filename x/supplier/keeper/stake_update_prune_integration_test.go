package keeper_test

import (
    "testing"

    cosmostypes "github.com/cosmos/cosmos-sdk/types"
    "github.com/stretchr/testify/require"

    keepertest "github.com/pokt-network/poktroll/testutil/keeper"
    "github.com/pokt-network/poktroll/testutil/sample"
    sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
    "github.com/pokt-network/poktroll/x/supplier/keeper"
)

// This integration-style test ensures that after a supplier restakes with a changed
// service set, services are still visible at the next session start and remain
// visible after EndBlocker pruning runs in the same block. This protects against
// operator-index inconsistencies by relying on the reindexing behavior in
// SetAndIndexDehydratedSupplier.
func TestStake_UpdateServices_ActivateAndPrune_PersistsServices(t *testing.T) {
    k, ctx := keepertest.SupplierKeeper(t)
    srv := keeper.NewMsgServerImpl(*k.Keeper)

    owner := sample.AccAddressBech32()
    operator := sample.AccAddressBech32()

    // Initial stake with one service "svc1"
    stakeMsg, _ := newSupplierStakeMsg(owner, operator, k.Keeper.GetParams(ctx).MinStake.Amount.Int64(), "svc1")
    _, err := srv.StakeSupplier(ctx, stakeMsg)
    require.NoError(t, err)

    // Move to next session start and activate services
    ctx = setBlockHeightToNextSessionStart(ctx, k.SharedKeeper)
    _, err = k.Keeper.BeginBlockerActivateSupplierServices(ctx)
    require.NoError(t, err)

    // Confirm svc1 active
    s, found := k.Keeper.GetSupplier(ctx, operator)
    require.True(t, found)
    require.Len(t, s.Services, 1)
    require.Equal(t, "svc1", s.Services[0].ServiceId)

    // Prepare update to replace services with "svc2"
    updateMsg, _ := newSupplierStakeMsg(owner, operator, 0, "svc2")
    updateMsg.Stake = nil // do not change stake
    setStakeMsgSigner(updateMsg, operator)

    _, err = srv.StakeSupplier(ctx, updateMsg)
    require.NoError(t, err)

    // Advance to the next session start where svc1 deactivates and svc2 activates
    sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
    sharedParams := k.SharedKeeper.GetParams(ctx)
    nextStart := sharedtypes.GetNextSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())
    ctx = keepertest.SetBlockHeight(ctx, nextStart)

    // Activate and then prune deactivated configs in the same block
    _, err = k.Keeper.BeginBlockerActivateSupplierServices(ctx)
    require.NoError(t, err)
    _, err = k.Keeper.EndBlockerPruneSupplierServiceConfigHistory(ctx)
    require.NoError(t, err)

    // Hydrated supplier must show only svc2 active
    s, found = k.Keeper.GetSupplier(ctx, operator)
    require.True(t, found)
    require.Len(t, s.Services, 1)
    require.Equal(t, "svc2", s.Services[0].ServiceId)
}
