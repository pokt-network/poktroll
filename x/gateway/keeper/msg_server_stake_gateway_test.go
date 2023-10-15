package keeper_test

import (
	"testing"

	keepertest "pocket/testutil/keeper"

	"pocket/testutil/sample"
	"pocket/x/gateway/keeper"
	"pocket/x/gateway/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgServer_StakeGateway_SuccessfulCreateAndUpdate(t *testing.T) {
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
	gateway := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway
	_, err := srv.StakeGateway(wctx, gateway)
	require.NoError(t, err)

	// Verify that the gateway exists
	foundGateway, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, initialStake.Amount, foundGateway.Stake.Amount)

	// Prepare an updated gateway with a higher stake
	updatedStake := sdk.NewCoin("upokt", sdk.NewInt(200))
	updatedGateway := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Update the staked gateway
	_, err = srv.StakeGateway(wctx, updatedGateway)
	require.NoError(t, err)
	foundGateway, isGatewayFound = k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, updatedStake.Amount, foundGateway.Stake.Amount)
}

func TestMsgServer_StakeGateway_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Prepare the gateway
	addr := sample.AccAddress()
	initialStake := sdk.NewCoin("upokt", sdk.NewInt(100))
	gateway := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway & verify that the gateway exists
	_, err := srv.StakeGateway(wctx, gateway)
	require.NoError(t, err)
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)

	// Prepare an updated gateway with a lower stake
	updatedStake := sdk.NewCoin("upokt", sdk.NewInt(50))
	updatedGateway := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Verify that it fails
	_, err = srv.StakeGateway(wctx, updatedGateway)
	require.Error(t, err)

	// Verify that the gateway stake is unchanged
	gatewayFound, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, initialStake.Amount, gatewayFound.Stake.Amount)
}
