package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestMsgServer_StakeGateway_SuccessfulCreateAndUpdate(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway
	initialStake := cosmostypes.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &gatewaytypes.MsgStakeGateway{
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

	// Prepare an updated gateway with a higher stake
	updatedStake := cosmostypes.NewCoin("upokt", math.NewInt(200))
	updateMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Update the staked gateway
	_, err = srv.StakeGateway(ctx, updateMsg)
	require.NoError(t, err)
	foundGateway, isGatewayFound = k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, updatedStake.Amount, foundGateway.Stake.Amount)
}

func TestMsgServer_StakeGateway_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Prepare the gateway
	addr := sample.AccAddress()
	initialStake := cosmostypes.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway & verify that the gateway exists
	_, err := srv.StakeGateway(ctx, stakeMsg)
	require.NoError(t, err)
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)

	// Prepare an updated gateway with a lower stake
	updatedStake := cosmostypes.NewCoin("upokt", math.NewInt(50))
	updateMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Verify that it fails
	_, err = srv.StakeGateway(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the gateway stake is unchanged
	gatewayFound, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, initialStake.Amount, gatewayFound.Stake.Amount)
}

func TestMsgServer_StakeGateway_FailBelowMinStake(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	addr := sample.AccAddress()
	gatewayStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)
	minStake := gatewayStake.AddAmount(math.NewInt(1))
	expectedErr := gatewaytypes.ErrGatewayInvalidStake.Wrapf("gateway %q must stake at least %s", addr, minStake)

	// Set the minimum stake to be greater than the gateway stake.
	err := k.SetParams(ctx, gatewaytypes.Params{
		MinStake: &minStake,
	})
	require.NoError(t, err)

	// Prepare the gateway
	stakeMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &gatewayStake,
	}

	// Stake the gateway & verify that the gateway exists
	_, err = srv.StakeGateway(ctx, stakeMsg)
	require.ErrorContains(t, err, expectedErr.Error())
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}
