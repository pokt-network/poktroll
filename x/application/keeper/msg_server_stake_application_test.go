package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
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
	addr := sample.AccAddress()

	// Verify that the app does not exist yet
	_, isAppFound := k.GetApplication(ctx, addr)
	require.False(t, isAppFound)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists
	appFound, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, addr, appFound.Address)
	require.Equal(t, int64(100), appFound.Stake.Amount.Int64())
	require.Len(t, appFound.ServiceConfigs, 1)
	require.Equal(t, "svc1", appFound.ServiceConfigs[0].Service.Id)

	// Prepare an updated application with a higher stake and another service
	updateStakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(200)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
			{
				Service: &sharedtypes.Service{Id: "svc2"},
			},
		},
	}

	// Update the staked application
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.NoError(t, err)
	appFound, isAppFound = k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, int64(200), appFound.Stake.Amount.Int64())
	require.Len(t, appFound.ServiceConfigs, 2)
	require.Equal(t, "svc1", appFound.ServiceConfigs[0].Service.Id)
	require.Equal(t, "svc2", appFound.ServiceConfigs[1].Service.Id)
}

func TestMsgServer_StakeApplication_FailRestakingDueToInvalidServices(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	appAddr := sample.AccAddress()

	// Prepare the application stake message
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the application stake message without any services
	updateStakeMsg := &types.MsgStakeApplication{
		Address:  appAddr,
		Stake:    &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the app still exists and is staked for svc1
	app, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, app.Address)
	require.Len(t, app.ServiceConfigs, 1)
	require.Equal(t, "svc1", app.ServiceConfigs[0].Service.Id)

	// Prepare the application stake message with an invalid service ID
	updateStakeMsg = &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1 INVALID ! & *"},
			},
		},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the app still exists and is staked for svc1
	app, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, app.Address)
	require.Len(t, app.ServiceConfigs, 1)
	require.Equal(t, "svc1", app.ServiceConfigs[0].Service.Id)
}

func TestMsgServer_StakeApplication_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Prepare the application
	addr := sample.AccAddress()
	stakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)

	// Prepare an updated application with a lower stake
	updateMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdkmath.NewInt(50)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Verify that it fails
	_, err = srv.StakeApplication(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the application stake is unchanged
	appFound, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, int64(100), appFound.Stake.Amount.Int64())
}
