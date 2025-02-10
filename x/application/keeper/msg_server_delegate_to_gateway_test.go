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
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
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
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr1, gatewaytypes.GatewayNotUnstaking)
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr2, gatewaytypes.GatewayNotUnstaking)

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

	expectedApp := &apptypes.Application{
		Address:                   stakeMsg.GetAddress(),
		Stake:                     stakeMsg.GetStake(),
		ServiceConfigs:            stakeMsg.GetServices(),
		DelegateeGatewayAddresses: make([]string, 0),
		PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.Equal(t, expectedApp, &foundApp)
	require.True(t, isAppFound)

	// Prepare the delegation message
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
	}

	expectedApp.DelegateeGatewayAddresses = append(expectedApp.DelegateeGatewayAddresses, gatewayAddr1)

	// Delegate the application to the gateway
	delegateRes, err := srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)
	require.Equal(t, delegateRes.GetApplication(), expectedApp)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	expectedEvent := &apptypes.EventRedelegation{
		Application:      expectedApp,
		SessionEndHeight: sessionEndHeight,
	}

	events := sdkCtx.EventManager().Events()
	filteredEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 1, len(filteredEvents), "expected exactly 1 EventRedelegation event")
	require.EqualValues(t, expectedEvent, filteredEvents[0])

	// Reset the events, as if a new block were created.
	ctx, sdkCtx = testevents.ResetEventManager(ctx)

	// Prepare a second delegation message
	delegateMsg2 := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the second gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg2)
	require.NoError(t, err)

	// Add gateway2 to the expected application's delegations.
	expectedApp.DelegateeGatewayAddresses = append(expectedApp.DelegateeGatewayAddresses, gatewayAddr2)

	events = sdkCtx.EventManager().Events()
	filteredEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 1, len(filteredEvents))
	require.EqualValues(t, expectedEvent, filteredEvents[0])

	// Verify that the application exists
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.EqualValues(t, expectedApp, &foundApp)
}

func TestMsgServer_DelegateToGateway_FailDuplicate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	// Mock the gateway being staked via the staked gateway map
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, gatewaytypes.GatewayNotUnstaking)

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
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	expectedApp := &apptypes.Application{
		Address:                   stakeMsg.GetAddress(),
		Stake:                     stakeMsg.GetStake(),
		ServiceConfigs:            stakeMsg.GetServices(),
		DelegateeGatewayAddresses: []string{gatewayAddr},
		PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
	}
	expectedEvent := &apptypes.EventRedelegation{
		Application:      expectedApp,
		SessionEndHeight: sessionEndHeight,
	}

	events := sdkCtx.EventManager().Events()
	filteredEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 1, len(filteredEvents))
	require.EqualValues(t, expectedEvent, filteredEvents[0])

	// Reset the events, as if a new block were created.
	ctx, sdkCtx = testevents.ResetEventManager(ctx)

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
	require.ErrorContains(t, err, apptypes.ErrAppAlreadyDelegated.Error())

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
	require.ErrorContains(t, err, apptypes.ErrAppGatewayNotFound.Error())
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
		keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, gatewaytypes.GatewayNotUnstaking)
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
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)

	events := sdkCtx.EventManager().Events()
	filteredEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, int(maxDelegatedParam), len(filteredEvents))

	for i, event := range filteredEvents {
		expectedApp := &apptypes.Application{
			Address:                   stakeMsg.GetAddress(),
			Stake:                     stakeMsg.GetStake(),
			ServiceConfigs:            stakeMsg.GetServices(),
			DelegateeGatewayAddresses: gatewayAddresses[:i+1],
			PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
		}
		expectedEvent := &apptypes.EventRedelegation{
			Application:      expectedApp,
			SessionEndHeight: sessionEndHeight,
		}
		require.EqualValues(t, expectedEvent, event)
	}

	// Reset the events, as if a new block were created.
	ctx, sdkCtx = testevents.ResetEventManager(ctx)

	// Generate an address for the gateway that'll exceed the max
	gatewayAddr := sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, gatewaytypes.GatewayNotUnstaking)

	// Prepare the delegation message
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application when the max is already reached
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.ErrorContains(t, err, apptypes.ErrAppMaxDelegatedGateways.Error())

	events = sdkCtx.EventManager().Events()
	filteredEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 0, len(filteredEvents), "expected no redelegation events")

	foundApp, isStakedAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isStakedAppFound)
	require.Equal(t, maxDelegatedParam, uint64(len(foundApp.DelegateeGatewayAddresses)))
}

func TestMsgServer_DelegateToGateway_FailGatewayInactive(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateway.
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1)

	currentHeight := sdkCtx.BlockHeight()
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)

	// Mock the gateway being staked and unbonding via the staked gateway map.
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, uint64(sessionEndHeight))

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

	// Stake the application & verify that the application exists.
	_, err := srv.StakeApplication(sdkCtx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(sdkCtx, appAddr)
	require.True(t, isAppFound)

	// Set the block height to the session end height + 1 to simulate the gateway becoming inactive.
	sdkCtx = sdkCtx.WithBlockHeight(sessionEndHeight + 1)

	// Prepare the delegation message
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the inactive gateway.
	_, err = srv.DelegateToGateway(sdkCtx, delegateMsg)
	require.ErrorContains(t, err, gatewaytypes.ErrGatewayIsInactive.Error())

	// Verify that the application is not delegated to the gateway.
	foundApp, isAppFound := k.GetApplication(sdkCtx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))
}
