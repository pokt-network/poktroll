package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_UndelegateFromGateway_SuccessfullyUndelegate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateways
	appAddr := sample.AccAddress()
	maxDelegatedGateways := k.GetParams(ctx).MaxDelegatedGateways
	gatewayAddresses := make([]string, int(maxDelegatedGateways))
	for i := 0; i < len(gatewayAddresses); i++ {
		gatewayAddr := sample.AccAddress()
		// Mock the gateway being staked via the staked gateway map
		keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)
		gatewayAddresses[i] = gatewayAddr
	}

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation messages and delegate the application to the gateways
	for _, gatewayAddr := range gatewayAddresses {
		delegateMsg := &types.MsgDelegateToGateway{
			AppAddress:     appAddr,
			GatewayAddress: gatewayAddr,
		}
		// Delegate the application to the gateway
		_, err = srv.DelegateToGateway(wctx, delegateMsg)
		require.NoError(t, err)
	}
	events := ctx.EventManager().Events()
	require.Equal(t, int(maxDelegatedGateways), len(events))
	for _, event := range events {
		require.Equal(t, "pocket.application.EventDelegateeChange", event.Type)
		require.Equal(t, "app_address", event.Attributes[0].Key)
		require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), event.Attributes[0].Value)
	}

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways, int64(len(foundApp.DelegateeGatewayAddresses)))
	for i, gatewayAddr := range gatewayAddresses {
		require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[i])
	}

	// Prepare an undelegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddresses[3],
	}

	// Undelegate the application from the gateway
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.NoError(t, err)
	events = ctx.EventManager().Events()
	require.Equal(t, int(maxDelegatedGateways)+1, len(events))
	require.Equal(t, "pocket.application.EventDelegateeChange", events[7].Type)
	require.Equal(t, "app_address", events[7].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[7].Attributes[0].Value)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways-1, int64(len(foundApp.DelegateeGatewayAddresses)))
	gatewayAddresses = append(gatewayAddresses[:3], gatewayAddresses[4:]...)
	for i, gatewayAddr := range gatewayAddresses {
		require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[i])
	}
}

func TestMsgServer_UndelegateFromGateway_FailNotDelegated(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr1 := sample.AccAddress()
	gatewayAddr2 := sample.AccAddress()
	// Mock the gateway being staked via the staked gateway map
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr1)
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr2)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the undelegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
	}

	// Attempt to undelgate the application from the gateway
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.ErrorIs(t, err, types.ErrAppNotDelegated)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))
	events := ctx.EventManager().Events()
	require.Equal(t, 0, len(events))

	// Prepare a delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(wctx, delegateMsg)
	require.NoError(t, err)
	events = ctx.EventManager().Events()
	require.Equal(t, 1, len(events))
	require.Equal(t, "pocket.application.EventDelegateeChange", events[0].Type)
	require.Equal(t, "app_address", events[0].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[0].Attributes[0].Value)

	// Ensure the failed undelegation did not affect the application
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.ErrorIs(t, err, types.ErrAppNotDelegated)
	events = ctx.EventManager().Events()
	require.Equal(t, 1, len(events))
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr2, foundApp.DelegateeGatewayAddresses[0])
}

func TestMsgServer_UndelegateFromGateway_SuccessfullyUndelegateFromUnstakedGateway(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateways
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	// Mock the gateway being staked via the staked gateway map
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message and delegate the application to the gateway
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}
	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(wctx, delegateMsg)
	require.NoError(t, err)
	events := ctx.EventManager().Events()
	require.Equal(t, 1, len(events))
	require.Equal(t, "pocket.application.EventDelegateeChange", events[0].Type)
	require.Equal(t, "app_address", events[0].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[0].Attributes[0].Value)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[0])

	// Mock unstaking the gateway
	keepertest.RemoveGatewayFromStakedGatewayMap(t, gatewayAddr)

	// Prepare an undelegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Undelegate the application from the gateway
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.NoError(t, err)
	events = ctx.EventManager().Events()
	require.Equal(t, 2, len(events))
	require.Equal(t, "pocket.application.EventDelegateeChange", events[1].Type)
	require.Equal(t, "app_address", events[1].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[1].Attributes[0].Value)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))
}
