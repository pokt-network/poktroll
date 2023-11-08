package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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
	initialStake := sdk.NewCoin("upokt", sdk.NewInt(100))
	stakeMsg := &types.MsgStakeApplication{
		Address: addr,
		Stake:   &initialStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists
	appFound, isAppFound := k.GetApplication(ctx, addr)
	require.True(t, isAppFound)
	require.Equal(t, addr, appFound.Address)
	require.Equal(t, initialStake.Amount, appFound.Stake.Amount)
	require.Len(t, appFound.ServiceConfigs, 1)

	// Unstake the application
	unstakeMsg := &types.MsgUnstakeApplication{Address: addr}
	_, err = srv.UnstakeApplication(wctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the app can no longer be found after unstaking
	_, isAppFound = k.GetApplication(ctx, addr)
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

	// Unstake the application
	unstakeMsg := &types.MsgUnstakeApplication{Address: addr}
	_, err := srv.UnstakeApplication(wctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrAppNotFound)

	_, isAppFound = k.GetApplication(ctx, addr)
	require.False(t, isAppFound)
}
