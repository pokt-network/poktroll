package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_StakeApplication_SuccessfulCreateAndUpdate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application
	appAddr := sample.AccAddress()

	// Verify that the app does not exist yet
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.False(t, isAppFound)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Stake the application
	stakeAppRes, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert that the response contains the staked application.
	app := stakeAppRes.GetApplication()
	require.Equal(t, stakeMsg.GetAddress(), app.GetAddress())
	require.Equal(t, stakeMsg.GetStake(), app.GetStake())
	require.Equal(t, stakeMsg.GetServices(), app.GetServiceConfigs())

	// Assert that the EventApplicationStaked event is emitted.
	expectedEvent, err := sdk.TypedEventToEvent(
		&types.EventApplicationStaked{
			AppAddress: stakeMsg.GetAddress(),
			Stake:      stakeMsg.GetStake(),
			Services:   stakeMsg.GetServices(),
		},
	)
	require.NoError(t, err)

	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = resetEvents(ctx)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].ServiceId)

	// Prepare an updated application with a higher stake and another service
	updateStakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(200)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
			{ServiceId: "svc2"},
		},
	}

	// Update the staked application
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, int64(200), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 2)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].ServiceId)
	require.Equal(t, "svc2", foundApp.ServiceConfigs[1].ServiceId)

	// Assert that the EventApplicationUpStaked event is emitted.
	expectedEvent, err = sdk.TypedEventToEvent(
		&types.EventApplicationUpStaked{
			AppAddress: updateStakeMsg.GetAddress(),
			Stake:      updateStakeMsg.GetStake(),
			Services:   updateStakeMsg.GetServices(),
		},
	)
	require.NoError(t, err)

	events = sdk.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equalf(t, 1, len(events), "expected exactly 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx = resetEvents(ctx)
}

func TestMsgServer_StakeApplication_FailRestakingDueToInvalidServices(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	appAddr := sample.AccAddress()

	// Prepare the application stake message
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the application stake message without any services
	updateStakeMsg := &types.MsgStakeApplication{
		Address:  appAddr,
		Stake:    &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the foundApp still exists and is staked for svc1
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].ServiceId)

	// Prepare the application stake message with an invalid service ID
	updateStakeMsg = &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1 INVALID ! & *"},
		},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the app still exists and is staked for svc1
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].ServiceId)
}

func TestMsgServer_StakeApplication_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Prepare the application
	appAddr := sample.AccAddress()
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare an updated application with a lower stake
	updateMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(50)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Verify that it fails
	_, err = srv.StakeApplication(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the application stake is unchanged
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
}

// resetEvents re-initializes the cosmos event manager in the context such that
// prior event emissions are erased.
func resetEvents(ctx context.Context) context.Context {
	return sdk.UnwrapSDKContext(ctx).WithEventManager(sdk.NewEventManager())
}
