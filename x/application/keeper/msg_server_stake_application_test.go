package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
	sharedtypes "pocket/x/shared/types"
)

func TestMsgServer_StakeApplication_SuccessfulCreateAndUpdate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application
	addr := sample.AccAddress()

	// Verify that the app does not exist yet
	_, isAppFound := k.GetApplication(ctx, addr)
	require.False(t, isAppFound)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, addr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].ServiceId.Id)

	// Prepare an updated application with a higher stake and another service
	updateStakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(200)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
			},
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc2"},
			},
		},
	}

	// Update the staked application
	_, err = srv.StakeApplication(wctx, updateStakeMsg)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, int64(200), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 2)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].ServiceId.Id)
	require.Equal(t, "svc2", foundApp.ServiceConfigs[1].ServiceId.Id)
}

func TestMsgServer_StakeApplication_FailRestakingDueToInvalidServices(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	appAddr := sample.AccAddress()

	// Prepare the application stake message
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the application stake message without any services
	updateStakeMsg := &types.MsgStakeApplication{
		Address:  appAddr,
		Stake:    &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(wctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the app still exists and is staked for svc1
	app, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, app.Address)
	require.Len(t, app.ServiceConfigs, 1)
	require.Equal(t, "svc1", app.ServiceConfigs[0].ServiceId.Id)

	// Prepare the application stake message with an invalid service ID
	updateStakeMsg = &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1 INVALID ! & *"},
			},
		},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(wctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the app still exists and is staked for svc1
	app, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, app.Address)
	require.Len(t, app.ServiceConfigs, 1)
	require.Equal(t, "svc1", app.ServiceConfigs[0].ServiceId.Id)
}

func TestMsgServer_StakeApplication_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Prepare the application
	addr := sample.AccAddress()
	stakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)

	// Prepare an updated application with a lower stake
	updateMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(50)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
			},
		},
	}

	// Verify that it fails
	_, err = srv.StakeApplication(wctx, updateMsg)
	require.Error(t, err)

	// Verify that the application stake is unchanged
	appFound, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, int64(100), appFound.Stake.Amount.Int64())
}
