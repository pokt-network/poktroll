package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var eventRedelegationTypeURL = sdk.MsgTypeURL(&apptypes.EventRedelegation{})

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
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
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
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	filteredEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events, eventRedelegationTypeURL)
	require.Equal(t, 1, len(filteredEvents), "expected exactly 1 EventRedelegation event")
	require.Equal(t, appAddr, filteredEvents[0].GetAppAddress())
	require.Equal(t, gatewayAddr1, filteredEvents[0].GetGatewayAddress())

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr1, foundApp.DelegateeGatewayAddresses[0])

	// Prepare a second delegation message
	delegateMsg2 := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the second gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg2)
	require.NoError(t, err)

	events = sdkCtx.EventManager().Events()
	filteredEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events, eventRedelegationTypeURL)
	require.Equal(t, 1, len(filteredEvents))
	require.Equal(t, appAddr, filteredEvents[0].GetAppAddress())
	require.Equal(t, gatewayAddr1, filteredEvents[0].GetGatewayAddress())

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
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
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
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()
	filteredEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events, eventRedelegationTypeURL)
	require.Equal(t, 1, len(filteredEvents))
	require.Equal(t, appAddr, filteredEvents[0].GetAppAddress())
	require.Equal(t, gatewayAddr, filteredEvents[0].GetGatewayAddress())

	// Reset the events, as if a new block were created.
	ctx = testevents.ResetEventManager(ctx)

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[0])

	// Prepare a second delegation message
	delegateMsg2 := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the gateway again
	_, err = srv.DelegateToGateway(ctx, delegateMsg2)
	require.ErrorIs(t, err, apptypes.ErrAppAlreadyDelegated)

	sdkCtx = sdk.UnwrapSDKContext(ctx)
	events = sdkCtx.EventManager().Events()
	require.Equal(t, 0, len(events))
}

func TestMsgServer_DelegateToGateway_FailGatewayNotStaked(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()

	// Prepare the application
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
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
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the unstaked gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.ErrorIs(t, err, apptypes.ErrAppGatewayNotFound)
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
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
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
		delegateMsg := &apptypes.MsgDelegateToGateway{
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
	filteredEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events, eventRedelegationTypeURL)
	require.Equal(t, int(maxDelegatedParam), len(filteredEvents))
	for i, event := range filteredEvents {
		require.Equal(t, appAddr, event.GetAppAddress())
		require.Equal(t, gatewayAddresses[i], event.GetGatewayAddress())
	}

	// Generate an address for the gateway that'll exceed the max
	gatewayAddr := sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)

	// Prepare the delegation message
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application when the max is already reached
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.ErrorIs(t, err, apptypes.ErrAppMaxDelegatedGateways)

	events = sdkCtx.EventManager().Events()
	filteredEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events, eventRedelegationTypeURL)
	require.Equal(t, int(maxDelegatedParam), len(filteredEvents))

	foundApp, isStakedAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isStakedAppFound)
	require.Equal(t, maxDelegatedParam, uint64(len(foundApp.DelegateeGatewayAddresses)))
}
