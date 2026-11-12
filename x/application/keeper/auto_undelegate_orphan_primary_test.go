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

// A delegation index entry whose PRIMARY application record is gone (an orphan)
// must be dropped by the auto-undelegate EndBlocker, not halt the chain. Such an
// orphan can no longer be created through messages now that non-canonical
// addresses are rejected, so it is planted here via the store-level behavior:
// a delegatee removed without an undelegation leaves its (gateway,app) entry, and
// removing the application then deletes the primary but not that stale entry.
func TestEndBlockerAutoUndelegate_OrphanedPrimaryDoesNotHalt(t *testing.T) {
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

	// Drop the delegatee from the app WITHOUT an undelegation record: SetApplication
	// leaves the (gateway,app) delegation index entry in place.
	app, found := k.GetApplication(ctx, appAddr)
	require.True(t, found)
	app.DelegateeGatewayAddresses = nil
	k.SetApplication(ctx, app)

	// Remove the application: its primary record is deleted, but the stale
	// (gateway,app) delegation entry survives -> orphan.
	k.RemoveApplication(ctx, app)
	iterator := k.GetDelegationsIterator(ctx, gatewayAddr)
	require.True(t, iterator.Valid(), "test setup: orphaned delegation entry with no primary")
	iterator.Close()

	// The gateway is unbonding+inactive, so the EndBlocker walks its delegations and
	// hits the orphan. It must drop it and continue, not return an error (= chain halt).
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(int64(sessionEndHeight) + 1)
	require.NoError(t, k.EndBlockerAutoUndelegateFromUnbondingGateways(sdkCtx))

	iterator = k.GetDelegationsIterator(sdkCtx, gatewayAddr)
	require.False(t, iterator.Valid(), "orphaned delegation entry must be removed")
	iterator.Close()
}
