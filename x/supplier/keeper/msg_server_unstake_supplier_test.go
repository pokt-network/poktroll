package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgServer_UnstakeSupplier_Success(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)

	initialStake := int64(100)
	stakeMsg := createStakeMsg(supplierAddr, initialStake)

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, supplierAddr, foundSupplier.Address)
	require.Equal(t, math.NewInt(initialStake), foundSupplier.Stake.Amount)
	require.Len(t, foundSupplier.Services, 1)

	// Initiate the supplier unstaking
	unstakeMsg := &types.MsgUnstakeSupplier{Address: supplierAddr}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the supplier entered the unbonding period
	foundSupplier, isSupplierFound = k.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Greater(t, foundSupplier.UnstakeCommitSessionEndHeight, int64(0))

	// Move block height to the end of the unbonding period
	unbondingHeight := foundSupplier.UnstakeCommitSessionEndHeight + int64(k.GetParams(ctx).SupplierUnbondingPeriodBlocks)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(unbondingHeight)

	// Run the endblocker to unbond the supplier
	err = k.EndBlockerUnbondSupplier(sdkCtx)
	require.NoError(t, err)

	// Make sure the supplier is removed from the suppliers list when the
	// unbonding period is over
	_, isSupplierFound = k.GetSupplier(sdkCtx, supplierAddr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_UnstakeSupplier_CancelUnbondingIfRestaked(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Stake the supplier
	initialStake := int64(100)
	stakeMsg := createStakeMsg(supplierAddr, initialStake)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the supplier exists with no unbonding height
	foundSupplier, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, int64(0), foundSupplier.UnstakeCommitSessionEndHeight)

	// Initiate the supplier unstaking
	unstakeMsg := &types.MsgUnstakeSupplier{Address: supplierAddr}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the supplier entered the unbonding period
	foundSupplier, isSupplierFound = k.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Greater(t, foundSupplier.UnstakeCommitSessionEndHeight, int64(0))

	unbondingHeight := foundSupplier.UnstakeCommitSessionEndHeight + int64(k.GetParams(ctx).SupplierUnbondingPeriodBlocks)

	// Stake the supplier again
	stakeMsg = createStakeMsg(supplierAddr, initialStake+1)
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Make sure the supplier is no longer in the unbonding period
	foundSupplier, isSupplierFound = k.GetSupplier(ctx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, foundSupplier.UnstakeCommitSessionEndHeight, int64(0))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(unbondingHeight)

	// Run the endblocker to unbond the supplier
	err = k.EndBlockerUnbondSupplier(sdkCtx)
	require.NoError(t, err)

	// Make sure the supplier is still in the suppliers list with an unbonding height of 0
	foundSupplier, isSupplierFound = k.GetSupplier(sdkCtx, supplierAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, foundSupplier.UnstakeCommitSessionEndHeight, int64(0))
}

func TestMsgServer_UnstakeSupplier_FailIfNotStaked(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)

	// Initiate the supplier unstaking
	unstakeMsg := &types.MsgUnstakeSupplier{Address: supplierAddr}
	_, err := srv.UnstakeSupplier(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrSupplierNotFound)

	_, isSupplierFound = k.GetSupplier(ctx, supplierAddr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_UnstakeSupplier_FailIfNotInUnbondingPeriod(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the supplier
	supplierAddr := sample.AccAddress()

	// Stake the supplier
	initialStake := int64(100)
	stakeMsg := createStakeMsg(supplierAddr, initialStake)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Initiate the supplier unstaking
	unstakeMsg := &types.MsgUnstakeSupplier{Address: supplierAddr}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	_, err = srv.UnstakeSupplier(sdkCtx, unstakeMsg)
	require.ErrorIs(t, err, types.ErrSupplierUnbonding)
}

func createStakeMsg(supplierAddr string, stakeAmount int64) *types.MsgStakeSupplier {
	initialStake := sdk.NewCoin("upokt", math.NewInt(stakeAmount))
	return &types.MsgStakeSupplier{
		Address: supplierAddr,
		Stake:   &initialStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: &sharedtypes.Service{
					Id: "svcId",
				},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     "http://localhost:8080",
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
		},
	}
}
