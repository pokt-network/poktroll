package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	"github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_UnstakeGateway_Success(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(sdkCtx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway
	initialStake := sdk.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway
	_, err := srv.StakeGateway(sdkCtx, stakeMsg)
	require.NoError(t, err)

	// Verify that the gateway exists
	foundGateway, isGatewayFound := k.GetGateway(sdkCtx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, initialStake.Amount, foundGateway.Stake.Amount)

	// Unstake the gateway
	unstakeMsg := &types.MsgUnstakeGateway{Address: addr}
	unstakeRes, err := srv.UnstakeGateway(sdkCtx, unstakeMsg)
	require.NoError(t, err)

	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := uint64(testsession.GetSessionEndHeightWithDefaultParams(currentHeight))
	expectedGateway := &types.Gateway{
		Address:                 foundGateway.Address,
		Stake:                   foundGateway.Stake,
		UnstakeSessionEndHeight: sessionEndHeight,
	}

	require.Equal(t, expectedGateway, unstakeRes.GetGateway())

	// Make sure the gateway is found after unstaking
	_, isGatewayFound = k.GetGateway(sdkCtx, addr)
	require.True(t, isGatewayFound)

	// Calculate the unbonding period blocks
	sharedParams := sharedtypes.DefaultParams()
	numBlocksPerSession := sharedParams.NumBlocksPerSession
	gatewayUnbondingPeriodSessions := sharedParams.GatewayUnbondingPeriodSessions
	unbondingPeriodBlocks := gatewayUnbondingPeriodSessions * numBlocksPerSession

	// Advance the block height to the gateway unbonding period sessions
	unbondingPeriodHeight := sessionEndHeight + unbondingPeriodBlocks
	sdkCtx = sdkCtx.WithBlockHeight(int64(unbondingPeriodHeight))

	k.EndBlockerUnbondGateways(sdkCtx)

	// Make sure the gateway can no longer be found after the unbonding period has elapsed.
	_, isGatewayFound = k.GetGateway(sdkCtx, addr)
	require.False(t, isGatewayFound)
}

func TestMsgServer_UnstakeGateway_FailIfAlreadyUnbonding(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway
	initialStake := sdk.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &types.MsgStakeGateway{
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
	unstakeMsg := &types.MsgUnstakeGateway{Address: addr}
	_, err = srv.UnstakeGateway(ctx, unstakeMsg)
	require.NoError(t, err)

	// Make sure the gateway is found after unstaking
	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)

	// Unstake the gateway again
	_, err = srv.UnstakeGateway(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorContains(t, err, types.ErrGatewayIsUnstaking.Error())

	// Make sure the gateway is found after unstaking
	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.True(t, isGatewayFound)
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
	unstakeMsg := &types.MsgUnstakeGateway{Address: addr}
	_, err := srv.UnstakeGateway(ctx, unstakeMsg)
	require.Error(t, err)
	require.ErrorContains(t, err, types.ErrGatewayNotFound.Error())

	_, isGatewayFound = k.GetGateway(ctx, addr)
	require.False(t, isGatewayFound)
}

func TestMsgServer_UnstakeGateway_RestakeBeforeUnbondingSuccess(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1)

	// Generate an address for the gateway
	addr := sample.AccAddress()

	// Verify that the gateway does not exist yet
	_, isGatewayFound := k.GetGateway(sdkCtx, addr)
	require.False(t, isGatewayFound)

	// Prepare the gateway
	initialStake := sdk.NewCoin("upokt", math.NewInt(100))
	stakeMsg := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	}

	// Stake the gateway
	_, err := srv.StakeGateway(sdkCtx, stakeMsg)
	require.NoError(t, err)

	// Verify that the gateway exists
	foundGateway, isGatewayFound := k.GetGateway(sdkCtx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, initialStake.Amount, foundGateway.Stake.Amount)

	// Unstake the gateway
	unstakeMsg := &types.MsgUnstakeGateway{Address: addr}
	unstakeRes, err := srv.UnstakeGateway(sdkCtx, unstakeMsg)
	require.NoError(t, err)

	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := uint64(testsession.GetSessionEndHeightWithDefaultParams(currentHeight))
	expectedGateway := &types.Gateway{
		Address:                 foundGateway.Address,
		Stake:                   foundGateway.Stake,
		UnstakeSessionEndHeight: sessionEndHeight,
	}

	require.Equal(t, expectedGateway, unstakeRes.Gateway)

	// Make sure the gateway is found after unstaking
	_, isGatewayFound = k.GetGateway(sdkCtx, addr)
	require.True(t, isGatewayFound)

	// Calculate the unbonding period blocks
	sharedParams := sharedtypes.DefaultParams()
	numBlocksPerSession := sharedParams.NumBlocksPerSession
	gatewayUnbondingPeriodSessions := sharedParams.GatewayUnbondingPeriodSessions
	unbondingPeriodBlocks := gatewayUnbondingPeriodSessions * numBlocksPerSession

	// Advance the block height to the gateway unbonding period sessions
	unbondingPeriodHeight := sessionEndHeight + unbondingPeriodBlocks
	sdkCtx = sdkCtx.WithBlockHeight(int64(unbondingPeriodHeight - 1))

	k.EndBlockerUnbondGateways(sdkCtx)

	// Make sure the gateway still exists before the unbonding period has elapsed.
	_, isGatewayFound = k.GetGateway(sdkCtx, addr)
	require.True(t, isGatewayFound)

	newStake := initialStake.AddAmount(math.NewInt(1))
	// Restake the gateway
	restakeMsg := &types.MsgStakeGateway{
		Address: addr,
		Stake:   &newStake,
	}

	_, err = srv.StakeGateway(sdkCtx, restakeMsg)
	require.NoError(t, err)

	// Verify that the gateway still exists
	foundGateway, isGatewayFound = k.GetGateway(sdkCtx, addr)
	require.True(t, isGatewayFound)
	require.Equal(t, addr, foundGateway.Address)
	require.Equal(t, &newStake, foundGateway.Stake)
	require.Equal(t, uint64(0), foundGateway.UnstakeSessionEndHeight)
}
