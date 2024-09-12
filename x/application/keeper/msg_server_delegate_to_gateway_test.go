package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
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
	srv := keeper.NewMsgServerImpl(k)

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
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	require.Equal(t, 1, len(events))
	require.Equal(t, "poktroll.application.EventRedelegation", events[0].Type)
	require.Equal(t, "app_address", events[0].Attributes[0].Key)
	require.Equal(t, "gateway_address", events[0].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[0].Attributes[0].Value)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddr1), events[0].Attributes[1].Value)

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
	_, err = srv.DelegateToGateway(ctx, delegateMsg2)
	require.NoError(t, err)

	events = sdkCtx.EventManager().Events()
	require.Equal(t, 2, len(events))
	require.Equal(t, "poktroll.application.EventRedelegation", events[1].Type)
	require.Equal(t, "app_address", events[1].Attributes[0].Key)
	require.Equal(t, "gateway_address", events[1].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[1].Attributes[0].Value)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddr2), events[1].Attributes[1].Value)

	// Verify that the application exists
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 2, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr1, foundApp.DelegateeGatewayAddresses[0])
	require.Equal(t, gatewayAddr2, foundApp.DelegateeGatewayAddresses[1])
}

func TestMsgServer_DelegateToGateway_FailDuplicate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	// Mock the gateway being staked via the staked gateway map
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	require.Equal(t, 1, len(events))
	require.Equal(t, "poktroll.application.EventRedelegation", events[0].Type)
	require.Equal(t, "app_address", events[0].Attributes[0].Key)
	require.Equal(t, "gateway_address", events[0].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[0].Attributes[0].Value)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddr), events[0].Attributes[1].Value)

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
	_, err = srv.DelegateToGateway(ctx, delegateMsg2)
	require.ErrorIs(t, err, types.ErrAppAlreadyDelegated)
	events = sdkCtx.EventManager().Events()
	require.Equal(t, 1, len(events))
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[0])
}

func TestMsgServer_DelegateToGateway_FailGatewayNotStaked(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the unstaked gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.ErrorIs(t, err, types.ErrAppGatewayNotFound)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))
}

func TestMsgServer_DelegateToGateway_FailMaxReached(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application
	appAddr := sample.AccAddress()

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isStakedAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isStakedAppFound)

	// Delegate the application to the max number of gateways
	maxDelegatedParam := k.GetParams(ctx).MaxDelegatedGateways
	gatewayAddresses := make([]string, maxDelegatedParam)
	for i := uint64(0); i < k.GetParams(ctx).MaxDelegatedGateways; i++ {
		// Prepare the delegation message
		gatewayAddr := sample.AccAddress()
		gatewayAddresses[i] = gatewayAddr
		// Mock the gateway being staked via the staked gateway map
		keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)
		delegateMsg := &types.MsgDelegateToGateway{
			AppAddress:     appAddr,
			GatewayAddress: gatewayAddr,
		}
		// Delegate the application to the gateway
		_, err = srv.DelegateToGateway(ctx, delegateMsg)
		require.NoError(t, err)
		// Check number of gateways delegated to is correct
		foundApp, isDelegatedAppFound := k.GetApplication(ctx, appAddr)
		require.True(t, isDelegatedAppFound)
		require.Equal(t, int(i+1), len(foundApp.DelegateeGatewayAddresses))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	require.Equal(t, int(maxDelegatedParam), len(events))
	for i, event := range events {
		require.Equal(t, "poktroll.application.EventRedelegation", event.Type)
		require.Equal(t, "app_address", event.Attributes[0].Key)
		require.Equal(t, "gateway_address", event.Attributes[1].Key)
		require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), event.Attributes[0].Value)
		require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddresses[i]), event.Attributes[1].Value)
	}

	// Generate an address for the gateway that'll exceed the max
	gatewayAddr := sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)

	// Prepare the delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application when the max is already reached
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.ErrorIs(t, err, types.ErrAppMaxDelegatedGateways)
	events = sdkCtx.EventManager().Events()
	require.Equal(t, int(maxDelegatedParam), len(events))
	foundApp, isStakedAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isStakedAppFound)
	require.Equal(t, maxDelegatedParam, uint64(len(foundApp.DelegateeGatewayAddresses)))
}
