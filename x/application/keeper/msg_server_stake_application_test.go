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
	initialStake := sdk.NewCoin("upokt", sdk.NewInt(100))
	app := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the application
	_, err := srv.StakeApplication(wctx, app)
	require.NoError(t, err)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, addr, foundApp.Address)
	require.Equal(t, initialStake.Amount, foundApp.Stake.Amount)

	// Prepare an updated application with a higher stake
	updatedStake := sdk.NewCoin("upokt", sdk.NewInt(200))
	updatedApp := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Update the staked application
	_, err = srv.StakeApplication(wctx, updatedApp)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, updatedStake.Amount, foundApp.Stake.Amount)
}

func TestMsgServer_StakeApplication_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Prepare the application
	addr := sample.AccAddress()
	initialStake := sdk.NewCoin("upokt", sdk.NewInt(100))
	app := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &stake,
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, app)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)

	// Prepare an updated application with a lower stake
	updatedStake := sdk.NewCoin("upokt", sdk.NewInt(50))
	updatedApp := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Verify that it fails
	_, err = srv.StakeApplication(wctx, updatedApp)
	require.Error(t, err)

	// Verify that the application stake is unchanged
	appFound, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, initialStake.Amount, appFound.Stake.Amount)
}
