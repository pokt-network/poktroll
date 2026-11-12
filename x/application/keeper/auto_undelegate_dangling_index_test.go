package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// A delegation index entry whose application no longer delegates to the
// unbonding gateway is removed by the auto-undelegation EndBlocker.
func TestEndBlockerAutoUndelegateFromUnbondingGateways_RemovesDanglingDelegationIndex(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	appAddr := sample.AccAddressBech32()
	_, err := srv.StakeApplication(ctx, &apptypes.MsgStakeApplication{
		Address:  appAddr,
		Stake:    &apptypes.DefaultMinStake,
		Services: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
	})
	require.NoError(t, err)

	sessionEndHeight := uint64(testsession.GetSessionEndHeightWithDefaultParams(1))
	gatewayAddr := sample.AccAddressBech32()
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, sessionEndHeight)
	_, err = srv.DelegateToGateway(ctx, &apptypes.MsgDelegateToGateway{AppAddress: appAddr, GatewayAddress: gatewayAddr})
	require.NoError(t, err)

	// Drop the delegation from the application without recording an undelegation:
	// SetApplication leaves the gateway's delegation index entry behind.
	app, isFound := k.GetApplication(ctx, appAddr)
	require.True(t, isFound)
	app.DelegateeGatewayAddresses = nil
	k.SetApplication(ctx, app)

	iterator := k.GetDelegationsIterator(ctx, gatewayAddr)
	require.True(t, iterator.Valid(), "test setup: dangling delegation index entry")
	iterator.Close()

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(int64(sessionEndHeight) + 1)
	require.NoError(t, k.EndBlockerAutoUndelegateFromUnbondingGateways(sdkCtx))

	iterator = k.GetDelegationsIterator(sdkCtx, gatewayAddr)
	require.False(t, iterator.Valid(), "dangling delegation index entry must be removed")
	iterator.Close()
}
