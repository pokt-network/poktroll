package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/gateway/keeper"
	"pocket/x/gateway/types"
)

func TestMsgServer_UnstakeGateway_Success(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway
	initialStake := sdk.NewCoin("upokt", sdk.NewInt(100))
	stakeMsg := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway
	_, err := srv.StakeGateway(wctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the gateway exists
	foundGateway, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, initialStake.Amount, foundGateway.Stake.Amount)

	// Unstake the gateway
	unstakeMsg := &types.MsgUnstakeGateway{Address: addr}
	_, err = srv.UnstakeGateway(wctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the gateway can no longer be found after unstaking
	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}

func TestMsgServer_UnstakeGateway_FailIfNotStaked(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Unstake the gateway
	unstakeMsg := &types.MsgUnstakeGateway{Address: addr}
	_, err := srv.UnstakeGateway(wctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrGatewayNotFound)

	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}
