package application

import (
	"testing"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// The unbonding end height reported when an application or a gateway unstakes
// is the height at which the EndBlocker releases it: it uses the shared params
// at the unstake session end, not the live params.
func TestUnbondingBeginEvents_UseParamsAtUnstakeSessionEnd(t *testing.T) {
	cmd.InitSDKConfig()

	keepers, ctx := keeper.NewTokenomicsModuleKeepers(t, cosmoslog.NewNopLogger())
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(5)

	// The params history has the default unbonding periods. The live params,
	// changed without a history entry, have longer ones.
	sharedKeeper := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	historyParams := sharedtypes.DefaultParams()
	require.NoError(t, sharedKeeper.SetParamsAtHeight(ctx, 1, historyParams))
	liveParams := historyParams
	liveParams.ApplicationUnbondingPeriodSessions += 3
	liveParams.GatewayUnbondingPeriodSessions += 3
	require.NoError(t, sharedKeeper.SetParams(ctx, liveParams))

	appKeeper := *keepers.ApplicationKeeper.(*appkeeper.Keeper)
	appAddr := sample.AccAddressBech32()
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address:        appAddr,
		Stake:          &apptypes.DefaultMinStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
	})

	gatewayAddr := sample.AccAddressBech32()
	gatewayStake := cosmostypes.NewInt64Coin("upokt", 1_000_000)
	keepers.GatewayKeeper.SetGateway(ctx, gatewaytypes.Gateway{Address: gatewayAddr, Stake: &gatewayStake})

	ctx, _ = testevents.ResetEventManager(ctx)
	_, err := appkeeper.NewMsgServerImpl(appKeeper).UnstakeApplication(ctx, &apptypes.MsgUnstakeApplication{Address: appAddr})
	require.NoError(t, err)
	_, err = gatewaykeeper.NewMsgServerImpl(*keepers.GatewayKeeper).UnstakeGateway(ctx, &gatewaytypes.MsgUnstakeGateway{Address: gatewayAddr})
	require.NoError(t, err)
	events := cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()

	app, found := appKeeper.GetApplication(ctx, appAddr)
	require.True(t, found)
	appEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingBegin](t, events)
	require.Len(t, appEvents, 1)
	require.Equal(t, apptypes.GetApplicationUnbondingHeight(&historyParams, &app), appEvents[0].GetUnbondingEndHeight())
	require.Equal(t, appKeeper.GetApplicationUnbondingEndHeight(ctx, &app), appEvents[0].GetUnbondingEndHeight())

	gateway, found := keepers.GatewayKeeper.GetGateway(ctx, gatewayAddr)
	require.True(t, found)
	gatewayEvents := testevents.FilterEvents[*gatewaytypes.EventGatewayUnbondingBegin](t, events)
	require.Len(t, gatewayEvents, 1)
	require.Equal(t, gatewaytypes.GetGatewayUnbondingHeight(&historyParams, &gateway), gatewayEvents[0].GetUnbondingEndHeight())
	require.Equal(t, keepers.GatewayKeeper.GetGatewayUnbondingEndHeight(ctx, &gateway), gatewayEvents[0].GetUnbondingEndHeight())
}
