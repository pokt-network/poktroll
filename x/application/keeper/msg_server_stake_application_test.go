package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
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
	stakeMsg := &application.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*shared.ApplicationServiceConfig{
			{
				Service: &shared.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)

	// Prepare an updated application with a higher stake and another service
	updateStakeMsg := &application.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(200)},
		Services: []*shared.ApplicationServiceConfig{
			{
				Service: &shared.Service{Id: "svc1"},
			},
			{
				Service: &shared.Service{Id: "svc2"},
			},
		},
	}

	// Update the staked application
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, int64(200), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 2)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)
	require.Equal(t, "svc2", foundApp.ServiceConfigs[1].Service.Id)
}

func TestMsgServer_StakeApplication_FailRestakingDueToInvalidServices(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	appAddr := sample.AccAddress()

	// Prepare the application stake message
	stakeMsg := &application.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*shared.ApplicationServiceConfig{
			{
				Service: &shared.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Prepare the application stake message without any services
	updateStakeMsg := &application.MsgStakeApplication{
		Address:  appAddr,
		Stake:    &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*shared.ApplicationServiceConfig{},
	}

	// Fail updating the application when the list of services is empty
	_, err = srv.StakeApplication(ctx, updateStakeMsg)
	require.Error(t, err)

	// Verify the foundApp still exists and is staked for svc1
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)

	// Prepare the application stake message with an invalid service ID
	updateStakeMsg = &application.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*shared.ApplicationServiceConfig{
			{
				Service: &shared.Service{Id: "svc1 INVALID ! & *"},
			},
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
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].Service.Id)
}

func TestMsgServer_StakeApplication_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Prepare the application
	appAddr := sample.AccAddress()
	stakeMsg := &application.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*shared.ApplicationServiceConfig{
			{
				Service: &shared.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare an updated application with a lower stake
	updateMsg := &application.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(50)},
		Services: []*shared.ApplicationServiceConfig{
			{
				Service: &shared.Service{Id: "svc1"},
			},
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
