package keeper_test

import (
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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
	app := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
	}

	// Stake the application
	_, err := srv.StakeApplication(wctx, app)
	require.NoError(t, err)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, addr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())

	// Prepare an updated application with a higher stake
	updatedApp := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(200)},
	}

	// Update the staked application
	_, err = srv.StakeApplication(wctx, updatedApp)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, int64(200), foundApp.Stake.Amount.Int64())
}

func TestMsgServer_StakeApplication_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Prepare the application
	addr := sample.AccAddress()
	app := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, app)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)

	// Prepare an updated application with a lower stake
	updatedApp := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(50)},
	}

	// Verify that it fails
	_, err = srv.StakeApplication(wctx, updatedApp)
	require.Error(t, err)

	// Verify that the application stake is unchanged
	appFound, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, int64(100), appFound.Stake.Amount.Int64())
}
