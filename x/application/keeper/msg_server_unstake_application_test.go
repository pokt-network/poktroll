package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
)

func TestMsgServer_UnstakeApplication_Success(t *testing.T) {
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
	updatedApp := &types.MsgUnstakeApplication{
		Address: addr,
	}

	// // Update the staked application
	_, err = srv.UnstakeApplication(wctx, updatedApp)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, addr)
	require.False(t, isAppFound)
}

func TestMsgServer_UnstakeApplication_FailIfNotStaked(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application
	addr := sample.AccAddress()

	// Verify that the app does not exist yet
	_, isAppFound := k.GetApplication(ctx, addr)
	require.False(t, isAppFound)

	// Prepare an updated application with a higher stake
	updatedApp := &types.MsgUnstakeApplication{
		Address: addr,
	}

	// Update the staked application
	_, err := srv.UnstakeApplication(wctx, updatedApp)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrAppNotFound)

	_, isAppFound = k.GetApplication(ctx, addr)
	require.False(t, isAppFound)
}
