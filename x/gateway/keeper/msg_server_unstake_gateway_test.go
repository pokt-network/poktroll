package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
)

func TestMsgServer_UnstakeGateway_Success(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway
	initialStake := sdk.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &gateway.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway
	_, err := srv.StakeGateway(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the gateway exists
	foundGateway, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, initialStake.Amount, foundGateway.Stake.Amount)

	// Unstake the gateway
	unstakeMsg := &gateway.MsgUnstakeGateway{Address: addr}
	_, err = srv.UnstakeGateway(ctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the gateway can no longer be found after unstaking
	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}

func TestMsgServer_UnstakeGateway_FailIfNotStaked(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Unstake the gateway
	unstakeMsg := &gateway.MsgUnstakeGateway{Address: addr}
	_, err := srv.UnstakeGateway(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, gateway.ErrGatewayNotFound)

	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}
