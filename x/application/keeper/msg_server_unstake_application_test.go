package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_UnstakeApplication_Success(t *testing.T) {
	applicationModuleKeepers, ctx := keepertest.NewApplicationModuleKeepers(t)
	srv := keeper.NewMsgServerImpl(*applicationModuleKeepers.Keeper)
	sharedParams := applicationModuleKeepers.SharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdk.UnwrapSDKContext(ctx).BlockHeight())

	// Generate an address for the application
	unstakingAppAddr := sample.AccAddress()

	// Verify that the app does not exist yet
	_, isAppFound := applicationModuleKeepers.GetApplication(ctx, unstakingAppAddr)
	require.False(t, isAppFound)

	// Prepare the application
	initialStake := apptypes.DefaultMinStake.Amount.Int64()
	stakeMsg := createAppStakeMsg(unstakingAppAddr, initialStake)

	// Stake the application
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists
	foundApp, isAppFound := applicationModuleKeepers.GetApplication(ctx, unstakingAppAddr)
	require.True(t, isAppFound)
	require.Equal(t, unstakingAppAddr, foundApp.Address)
	require.Equal(t, initialStake, foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)

	// Create and stake another application that will not be unstaked to assert that
	// only the unstaking application is removed from the applications list when the
	// unbonding period is over.
	nonUnstakingAppAddr := sample.AccAddress()
	stakeMsg = createAppStakeMsg(nonUnstakingAppAddr, initialStake)
	_, err = srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the non-unstaking application exists
	_, isAppFound = applicationModuleKeepers.GetApplication(ctx, nonUnstakingAppAddr)
	require.True(t, isAppFound)

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Unstake the application
	unstakeMsg := &apptypes.MsgUnstakeApplication{Address: unstakingAppAddr}
	_, err = srv.UnstakeApplication(ctx, unstakeMsg)
	require.NoError(t, err)

	// Assert that the EventApplicationStaked event is emitted.
	foundApp.UnstakeSessionEndHeight = uint64(sessionEndHeight)
	expectedEvent, err := sdk.TypedEventToEvent(
		&apptypes.EventApplicationUnbondingBegin{
			Application:      &foundApp,
			Reason:           apptypes.ApplicationUnbondingReason_ELECTIVE,
			SessionEndHeight: sessionEndHeight,
		},
	)
	require.NoError(t, err)

	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Make sure the application entered the unbonding period
	foundApp, isAppFound = applicationModuleKeepers.GetApplication(ctx, unstakingAppAddr)
	require.True(t, isAppFound)
	require.True(t, foundApp.IsUnbonding())

	// Move block height to the end of the unbonding period
	unbondingHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &foundApp)
	ctx = keepertest.SetBlockHeight(ctx, unbondingHeight)
	unbondingSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, unbondingHeight)

	// Run the endblocker to unbond applications
	err = applicationModuleKeepers.EndBlockerUnbondApplications(ctx)
	require.NoError(t, err)

	// Assert that the EventApplicationStaked event is emitted.
	expectedEvent, err = sdk.TypedEventToEvent(
		&apptypes.EventApplicationUnbondingEnd{
			Application:      &foundApp,
			Reason:           apptypes.ApplicationUnbondingReason_ELECTIVE,
			SessionEndHeight: unbondingSessionEndHeight,
		},
	)
	require.NoError(t, err)

	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Make sure the unstaking application is removed from the applications list when
	// the unbonding period is over.
	_, isAppFound = applicationModuleKeepers.GetApplication(ctx, unstakingAppAddr)
	require.False(t, isAppFound)

	// Verify that the non-unstaking application still exists.
	nonUnstakingApplication, isAppFound := applicationModuleKeepers.GetApplication(ctx, nonUnstakingAppAddr)
	require.True(t, isAppFound)
	require.False(t, nonUnstakingApplication.IsUnbonding())
}

func TestMsgServer_UnstakeApplication_CancelUnbondingIfRestaked(t *testing.T) {
	applicationModuleKeepers, ctx := keepertest.NewApplicationModuleKeepers(t)
	srv := keeper.NewMsgServerImpl(*applicationModuleKeepers.Keeper)
	sharedParams := applicationModuleKeepers.SharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdk.UnwrapSDKContext(ctx).BlockHeight())

	// Generate an address for the application
	appAddr := sample.AccAddress()

	// Stake the application
	initialStake := apptypes.DefaultMinStake.Amount.Int64()
	stakeMsg := createAppStakeMsg(appAddr, initialStake)
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists with no unbonding height
	foundApp, isAppFound := applicationModuleKeepers.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.False(t, foundApp.IsUnbonding())

	// Initiate the application unstaking
	unstakeMsg := &apptypes.MsgUnstakeApplication{Address: appAddr}
	_, err = srv.UnstakeApplication(ctx, unstakeMsg)
	require.NoError(t, err)

	// Assert that the EventApplicationUnbondingCanceled event is emitted.
	foundApp.UnstakeSessionEndHeight = uint64(sessionEndHeight)
	foundApp.DelegateeGatewayAddresses = make([]string, 0)
	expectedAppUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
		Application:      &foundApp,
		Reason:           apptypes.ApplicationUnbondingReason_ELECTIVE,
		SessionEndHeight: sessionEndHeight,
	}
	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	appUnbondingBeginEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingBegin](t, events)
	require.Equalf(t, 1, len(appUnbondingBeginEvents), "expected exactly 1 event")
	require.EqualValues(t, expectedAppUnbondingBeginEvent, appUnbondingBeginEvents[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Make sure the application entered the unbonding period
	foundApp, isAppFound = applicationModuleKeepers.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.True(t, foundApp.IsUnbonding())

	unbondingHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &foundApp)

	// Stake the application again
	stakeMsg = createAppStakeMsg(appAddr, initialStake+1)
	stakeRes, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert that the EventApplicationUnbondingCanceled event is emitted.
	expectedApp := stakeRes.GetApplication()
	expectedApp.DelegateeGatewayAddresses = make([]string, 0)
	expectedAppUnbondingCanceledEvent := &apptypes.EventApplicationUnbondingCanceled{Application: expectedApp}
	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	appUnbondingEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingCanceled](t, events)
	require.Equalf(t, 1, len(appUnbondingEvents), "expected exactly 1 event")
	require.EqualValues(t, expectedAppUnbondingCanceledEvent, appUnbondingEvents[0])

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Make sure the application is no longer in the unbonding period
	foundApp, isAppFound = applicationModuleKeepers.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.False(t, foundApp.IsUnbonding())

	ctx = keepertest.SetBlockHeight(ctx, unbondingHeight)

	// Run the EndBlocker, the application should not be unbonding.
	err = applicationModuleKeepers.EndBlockerUnbondApplications(ctx)
	require.NoError(t, err)

	// Make sure the application exists with an unbonding height of 0
	foundApp, isAppFound = applicationModuleKeepers.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.False(t, foundApp.IsUnbonding())
}

func TestMsgServer_UnstakeApplication_FailIfNotStaked(t *testing.T) {
	applicationModuleKeepers, ctx := keepertest.NewApplicationModuleKeepers(t)
	srv := keeper.NewMsgServerImpl(*applicationModuleKeepers.Keeper)

	// Generate an address for the application
	appAddr := sample.AccAddress()

	// Verify that the app does not exist yet
	_, isAppFound := applicationModuleKeepers.GetApplication(ctx, appAddr)
	require.False(t, isAppFound)

	// Unstake the application
	unstakeMsg := &apptypes.MsgUnstakeApplication{Address: appAddr}
	_, err := srv.UnstakeApplication(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, apptypes.ErrAppNotFound)

	_, isAppFound = applicationModuleKeepers.GetApplication(ctx, appAddr)
	require.False(t, isAppFound)
}

func TestMsgServer_UnstakeApplication_FailIfCurrentlyUnstaking(t *testing.T) {
	applicationModuleKeepers, ctx := keepertest.NewApplicationModuleKeepers(t)
	srv := keeper.NewMsgServerImpl(*applicationModuleKeepers.Keeper)

	// Generate an address for the application
	appAddr := sample.AccAddress()

	// Stake the application
	initialStake := apptypes.DefaultMinStake.Amount.Int64()
	stakeMsg := createAppStakeMsg(appAddr, initialStake)
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Initiate the application unstaking
	unstakeMsg := &apptypes.MsgUnstakeApplication{Address: appAddr}
	_, err = srv.UnstakeApplication(ctx, unstakeMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = keepertest.SetBlockHeight(ctx, sdkCtx.BlockHeight()+1)

	// Verify that the application cannot unstake if it is already unstaking.
	_, err = srv.UnstakeApplication(ctx, unstakeMsg)
	require.ErrorIs(t, err, apptypes.ErrAppIsUnstaking)
}

func createAppStakeMsg(appAddr string, stakeAmount int64) *apptypes.MsgStakeApplication {
	initialStake := sdk.NewInt64Coin("upokt", stakeAmount)

	return &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &initialStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}
}
