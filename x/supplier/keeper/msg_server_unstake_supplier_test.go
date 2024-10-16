package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const serviceID = "svcId"

func TestMsgServer_UnstakeSupplier_Success(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)

	// Generate an operator addresses for a supplier that will be unstaked later in the test.
	unstakingSupplierOperatorAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, unstakingSupplierOperatorAddr)
	require.False(t, isSupplierFound)

	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, expectedSupplier := newSupplierStakeMsg(unstakingSupplierOperatorAddr, unstakingSupplierOperatorAddr, initialStake, serviceID)

	// Stake the supplier
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert that the EventSupplierStaked event is emitted.
	expectedEvent, err := sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierStaked{Supplier: expectedSupplier},
	)
	require.NoError(t, err)

	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Verify that the supplier exists
	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, unstakingSupplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.Equal(t, unstakingSupplierOperatorAddr, foundSupplier.OperatorAddress)
	require.Equal(t, math.NewInt(initialStake), foundSupplier.Stake.Amount)
	require.Len(t, foundSupplier.Services, 1)

	// Create and stake another supplier that will not be unstaked to assert that only the
	// unstaking supplier is removed from the suppliers list when the unbonding period is over.
	nonUnstakingSupplierOperatorAddr := sample.AccAddress()
	stakeMsg, _ = newSupplierStakeMsg(nonUnstakingSupplierOperatorAddr, nonUnstakingSupplierOperatorAddr, initialStake, serviceID)
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Verify that the non-unstaking supplier exists
	_, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, nonUnstakingSupplierOperatorAddr)
	require.True(t, isSupplierFound)

	// Initiate the supplier unstaking
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          unstakingSupplierOperatorAddr,
		OperatorAddress: unstakingSupplierOperatorAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	expectedSupplier.UnstakeSessionEndHeight = uint64(shared.GetSessionEndHeight(&sharedParams, sdk.UnwrapSDKContext(ctx).BlockHeight()))
	unbondingHeight := shared.GetSupplierUnbondingHeight(&sharedParams, expectedSupplier)

	// Assert that the EventSupplierUnbondingCanceled event is emitted.
	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")

	expectedEvent, err = sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierUnbondingBegin{
			Supplier:        expectedSupplier,
			UnbondingHeight: unbondingHeight,
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Make sure the supplier entered the unbonding period
	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, unstakingSupplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.True(t, foundSupplier.IsUnbonding())

	// Move block height to the end of the unbonding period
	ctx = keepertest.SetBlockHeight(ctx, unbondingHeight)

	// Run the endblocker to unbond suppliers
	err = supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)

	// Assert that the EventSupplierUnbondingCanceled event is emitted.
	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 2 event")

	expectedEvent, err = sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierUnbondingEnd{
			Supplier:        expectedSupplier,
			UnbondingHeight: unbondingHeight,
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, expectedEvent, events[0])

	// Make sure the unstaking supplier is removed from the suppliers list when the
	// unbonding period is over
	_, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, unstakingSupplierOperatorAddr)
	require.False(t, isSupplierFound)

	// Verify that the non-unstaking supplier still exists
	nonUnstakingSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, nonUnstakingSupplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.False(t, nonUnstakingSupplier.IsUnbonding())
}

func TestMsgServer_UnstakeSupplier_CancelUnbondingIfRestaked(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)

	// Generate an address for the supplier
	supplierOperatorAddr := sample.AccAddress()

	// Stake the supplier
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, expectedSupplier := newSupplierStakeMsg(supplierOperatorAddr, supplierOperatorAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert that the EventSupplierStaked event is emitted.
	expectedEvent, err := sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierStaked{Supplier: expectedSupplier},
	)
	require.NoError(t, err)

	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Verify that the supplier exists with no unbonding height
	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.False(t, foundSupplier.IsUnbonding())

	// Initiate the supplier unstaking
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierOperatorAddr,
		OperatorAddress: supplierOperatorAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	expectedSupplier.UnstakeSessionEndHeight = uint64(shared.GetSessionEndHeight(&sharedParams, sdk.UnwrapSDKContext(ctx).BlockHeight()))
	unbondingHeight := sharedtypes.GetSupplierUnbondingHeight(&sharedParams, expectedSupplier)

	// Assert that the EventSupplierUnbondingBegin event is emitted.
	expectedEvent, err = sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierUnbondingBegin{
			Supplier:        expectedSupplier,
			UnbondingHeight: unbondingHeight,
		},
	)
	require.NoError(t, err)

	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Make sure the supplier entered the unbonding period
	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.True(t, foundSupplier.IsUnbonding())

	// Stake the supplier again
	stakeMsg, _ = newSupplierStakeMsg(supplierOperatorAddr, supplierOperatorAddr, initialStake+1, serviceID)
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	expectedSupplier.UnstakeSessionEndHeight = sharedtypes.SupplierNotUnstaking
	expectedSupplier.Stake = stakeMsg.GetStake()

	// Assert that the EventSupplierUnbondingCanceled event is emitted.
	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 2, len(events), "expected exactly 2 event")

	expectedEvent, err = sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierUnbondingCanceled{Supplier: expectedSupplier},
	)
	require.NoError(t, err)
	require.EqualValues(t, expectedEvent, events[0])

	expectedEvent, err = sdk.TypedEventToEvent(
		&suppliertypes.EventSupplierStaked{Supplier: expectedSupplier},
	)
	require.NoError(t, err)
	require.EqualValues(t, expectedEvent, events[1])

	// Make sure the supplier is no longer in the unbonding period
	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.False(t, foundSupplier.IsUnbonding())

	ctx = keepertest.SetBlockHeight(ctx, unbondingHeight)

	// Run the EndBlocker, the supplier should not be unbonding.
	err = supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)

	// Make sure the supplier is still in the suppliers list with an unbonding height of 0
	foundSupplier, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.False(t, foundSupplier.IsUnbonding())
}

func TestMsgServer_UnstakeSupplier_FailIfNotStaked(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an address for the supplier
	supplierOperatorAddr := sample.AccAddress()

	// Verify that the supplier does not exist yet
	_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.False(t, isSupplierFound)

	// Initiate the supplier unstaking
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierOperatorAddr,
		OperatorAddress: supplierOperatorAddr,
	}
	_, err := srv.UnstakeSupplier(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, suppliertypes.ErrSupplierNotFound)

	_, isSupplierFound = supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.False(t, isSupplierFound)
}

func TestMsgServer_UnstakeSupplier_FailIfCurrentlyUnstaking(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an address for the supplier
	supplierOperatorAddr := sample.AccAddress()

	// Stake the supplier
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(supplierOperatorAddr, supplierOperatorAddr, initialStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Initiate the supplier unstaking
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierOperatorAddr,
		OperatorAddress: supplierOperatorAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = keepertest.SetBlockHeight(ctx, sdkCtx.BlockHeight()+1)

	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.ErrorIs(t, err, suppliertypes.ErrSupplierIsUnstaking)
}

func TestMsgServer_UnstakeSupplier_OperatorCanUnstake(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	// Generate an address for the supplier
	ownerAddr := sample.AccAddress()
	supplierOperatorAddr := sample.AccAddress()

	// Stake the supplier
	initialStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, ownerAddr, initialStake, serviceID)
	stakeMsg.OperatorAddress = supplierOperatorAddr
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Initiate the supplier unstaking
	unstakeMsg := &suppliertypes.MsgUnstakeSupplier{
		Signer:          supplierOperatorAddr,
		OperatorAddress: supplierOperatorAddr,
	}
	_, err = srv.UnstakeSupplier(ctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the supplier entered the unbonding period
	foundSupplier, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, isSupplierFound)
	require.True(t, foundSupplier.IsUnbonding())

	// Move block height to the end of the unbonding period
	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	unbondingHeight := sharedtypes.GetSupplierUnbondingHeight(&sharedParams, &foundSupplier)
	ctx = keepertest.SetBlockHeight(ctx, int64(unbondingHeight))

	// Ensure that the initial stake is not returned to the owner yet
	require.Equal(t, int64(0), supplierModuleKeepers.SupplierUnstakedFundsMap[ownerAddr])

	// Run the endblocker to unbond suppliers
	err = supplierModuleKeepers.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)

	// Ensure that the initial stake is returned to the owner
	require.Equal(t, initialStake, supplierModuleKeepers.SupplierUnstakedFundsMap[ownerAddr])
}

func createStakeMsg(supplierOwnerAddr string, stakeAmount int64) *suppliertypes.MsgStakeSupplier {
	initialStake := sdk.NewCoin("upokt", math.NewInt(stakeAmount))
	return &suppliertypes.MsgStakeSupplier{
		Signer:          supplierOwnerAddr,
		OwnerAddress:    supplierOwnerAddr,
		OperatorAddress: supplierOwnerAddr,
		Stake:           &initialStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: "svcId",
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     "http://localhost:8080",
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            supplierOwnerAddr,
						RevSharePercentage: 100,
					},
				},
			},
		},
	}
}
