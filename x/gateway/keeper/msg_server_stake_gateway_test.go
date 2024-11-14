package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_StakeGateway_SuccessfulCreateAndUpdate(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the gateway.
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet.
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway.
	initialStake := cosmostypes.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway.
	stakeGatewayRes, err := srv.StakeGateway(ctx, stakeMsg)
	require.NoError(t, err)

	// Assert that the response contains the staked gateway.
	gateway := stakeGatewayRes.GetGateway()
	require.Equal(t, stakeMsg.GetAddress(), gateway.GetAddress())
	require.Equal(t, stakeMsg.GetStake(), gateway.GetStake())

	// Assert that the EventGatewayStaked event is emitted.
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	expectedEvent, err := cosmostypes.TypedEventToEvent(
		&gatewaytypes.EventGatewayStaked{
			Gateway:          gateway,
			SessionEndHeight: sessionEndHeight,
		},
	)
	require.NoError(t, err)

	events := cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equal(t, 1, len(events), "expected 1 event")
	require.EqualValues(t, expectedEvent, events[0])

	// Reset the events, as if a new block were created.
	ctx, _ = testevents.ResetEventManager(ctx)

	// Verify that the gateway exists.
	foundGateway, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, initialStake.Amount, foundGateway.Stake.Amount)

	// Prepare an updated gateway with a higher stake.
	updatedStake := cosmostypes.NewCoin("upokt", math.NewInt(200))
	upStakeMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Update the staked gateway.
	stakeGatewayRes, err = srv.StakeGateway(ctx, upStakeMsg)
	require.NoError(t, err)
	foundGateway, isGatewayFound = k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, updatedStake.Amount, foundGateway.Stake.Amount)

	// Assert that the response contains the upstaked gateway.
	upStakedGateway := stakeGatewayRes.GetGateway()
	require.Equal(t, upStakeMsg.GetAddress(), upStakedGateway.GetAddress())
	require.Equal(t, upStakeMsg.GetStake(), upStakedGateway.GetStake())

	// Assert that the EventGatewayStaked event is emitted.
	expectedEvent, err = cosmostypes.TypedEventToEvent(
		&gatewaytypes.EventGatewayStaked{
			Gateway:          upStakedGateway,
			SessionEndHeight: sessionEndHeight,
		},
	)
	require.NoError(t, err)

	events = cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()
	require.Equal(t, 1, len(events), "expected 1 event")
	require.EqualValues(t, expectedEvent, events[0])
}

func TestMsgServer_StakeGateway_FailLoweringStake(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Prepare the gateway.
	addr := sample.AccAddress()
	initialStake := cosmostypes.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway & verify that the gateway exists.
	_, err := srv.StakeGateway(ctx, stakeMsg)
	require.NoError(t, err)
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)

	// Prepare an updated gateway with a lower stake.
	updatedStake := cosmostypes.NewCoin("upokt", math.NewInt(50))
	updateMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &updatedStake,
	}

	// Verify that it fails.
	_, err = srv.StakeGateway(ctx, updateMsg)
	require.Error(t, err)

	// Verify that the gateway stake is unchanged.
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
	params := k.GetParams(ctx)
	params.MinStake = &minStake
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Prepare the gateway.
	stakeMsg := &gatewaytypes.MsgStakeGateway{
		Address: addr,
		Stake:   &gatewayStake,
	}

	// Attempt to stake the gateway & verify that the gateway does NOT exist.
	_, err = srv.StakeGateway(ctx, stakeMsg)
	require.ErrorContains(t, err, expectedErr.Error())
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}
