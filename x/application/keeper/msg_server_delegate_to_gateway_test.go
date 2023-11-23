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

func TestMsgServer_DelegateToGateway_SuccessfullyDelegate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateways
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

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
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
	require.Equal(t, gatewayAddr1, foundApp.DelegateeGatewayAddresses[0])

	// Prepare a second delegation message
	delegateMsg2 := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the second gateway
	_, err = srv.DelegateToGateway(wctx, delegateMsg2)
	require.NoError(t, err)
	events = ctx.EventManager().Events()
	require.Equal(t, 2, len(events))
	require.Equal(t, "pocket.application.EventDelegateeChange", events[1].Type)
	require.Equal(t, "app_address", events[1].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[1].Attributes[0].Value)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 2, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr1, foundApp.DelegateeGatewayAddresses[0])
	require.Equal(t, gatewayAddr2, foundApp.DelegateeGatewayAddresses[1])
}

func TestMsgServer_DelegateToGateway_FailDuplicate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateway
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

	// Prepare the delegation message
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

	// Prepare a second delegation message
	delegateMsg2 := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the gateway again
	_, err = srv.DelegateToGateway(wctx, delegateMsg2)
	require.ErrorIs(t, err, types.ErrAppAlreadyDelegated)
	events = ctx.EventManager().Events()
	require.Equal(t, 1, len(events))
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[0])
}

func TestMsgServer_DelegateToGateway_FailGatewayNotStaked(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()

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

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the unstaked gateway
	_, err = srv.DelegateToGateway(wctx, delegateMsg)
	require.ErrorIs(t, err, types.ErrAppGatewayNotFound)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))
}

func TestMsgServer_DelegateToGateway_FailMaxReached(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateway
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

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Delegate the application to the max number of gateways
	maxDelegatedParam := k.GetParams(ctx).MaxDelegatedGateways
	for i := int64(0); i < k.GetParams(ctx).MaxDelegatedGateways; i++ {
		// Prepare the delegation message
		gatewayAddr := sample.AccAddress()
		// Mock the gateway being staked via the staked gateway map
		keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)
		delegateMsg := &types.MsgDelegateToGateway{
			AppAddress:     appAddr,
			GatewayAddress: gatewayAddr,
		}
		// Delegate the application to the gateway
		_, err = srv.DelegateToGateway(wctx, delegateMsg)
		require.NoError(t, err)
		// Check number of gateways delegated to is correct
		foundApp, isAppFound := k.GetApplication(ctx, appAddr)
		require.True(t, isAppFound)
		require.Equal(t, int(i+1), len(foundApp.DelegateeGatewayAddresses))
	}
	events := ctx.EventManager().Events()
	require.Equal(t, int(maxDelegatedParam), len(events))
	for _, event := range events {
		require.Equal(t, "pocket.application.EventDelegateeChange", event.Type)
		require.Equal(t, "app_address", event.Attributes[0].Key)
		require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), event.Attributes[0].Value)
	}

	// Attempt to delegate the application when the max is already reached
	_, err = srv.DelegateToGateway(wctx, delegateMsg)
	require.ErrorIs(t, err, types.ErrAppMaxDelegatedGateways)
	events = ctx.EventManager().Events()
	require.Equal(t, int(maxDelegatedParam), len(events))
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, maxDelegatedParam, int64(len(foundApp.DelegateeGatewayAddresses)))
}
