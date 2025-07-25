package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gwtypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_UndelegateFromGateway_SuccessfullyUndelegate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateways
	appAddr := sample.AccAddress()
	maxDelegatedGateways := k.GetParams(ctx).MaxDelegatedGateways
	expectedGatewayAddresses := make([]string, int(maxDelegatedGateways))
	for i := 0; i < len(expectedGatewayAddresses); i++ {
		gatewayAddr := sample.AccAddress()
		// Mock the gateway being staked via the staked gateway map
		keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, 0)
		expectedGatewayAddresses[i] = gatewayAddr
	}

	// Prepare the application
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation messages and delegate the application to the gateways
	for _, gatewayAddr := range expectedGatewayAddresses {
		delegateMsg := &apptypes.MsgDelegateToGateway{
			AppAddress:     appAddr,
			GatewayAddress: gatewayAddr,
		}
		// Delegate the application to the gateway
		_, err = srv.DelegateToGateway(ctx, delegateMsg)
		require.NoError(t, err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)

	// Assert that the EventRedelegation events are emitted.
	events := sdkCtx.EventManager().Events()
	redelgationEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, int(maxDelegatedGateways), len(redelgationEvents))

	for i, redelegationEvent := range redelgationEvents {
		expectedApp := &apptypes.Application{
			Address:                   stakeMsg.GetAddress(),
			Stake:                     stakeMsg.GetStake(),
			ServiceConfigs:            stakeMsg.GetServices(),
			DelegateeGatewayAddresses: expectedGatewayAddresses[:i+1],
			PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
		}
		expectedRedelegationEvent := &apptypes.EventRedelegation{
			Application:      expectedApp,
			SessionEndHeight: sessionEndHeight,
		}
		require.EqualValues(t, expectedRedelegationEvent, redelegationEvent)
	}

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways, uint64(len(foundApp.DelegateeGatewayAddresses)))

	for i, gatewayAddr := range expectedGatewayAddresses {
		require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[i])
	}

	// Prepare an undelegation message
	undelegateMsg := &apptypes.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: expectedGatewayAddresses[3],
	}

	// Assert that the EventRedelegation event is emitted.
	expectedGatewayAddresses = append(expectedGatewayAddresses[:3], expectedGatewayAddresses[4:]...)
	expectedApp := &apptypes.Application{
		Address:                   stakeMsg.GetAddress(),
		Stake:                     stakeMsg.GetStake(),
		ServiceConfigs:            stakeMsg.GetServices(),
		DelegateeGatewayAddresses: expectedGatewayAddresses,
		PendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
			uint64(sessionEndHeight): {GatewayAddresses: []string{undelegateMsg.GetGatewayAddress()}},
		},
	}

	// Undelegate the application from the gateway
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.NoError(t, err)

	// Query the updated application from the keeper
	updatedApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, expectedApp, &updatedApp)

	events = sdkCtx.EventManager().Events()
	redelgationEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	lastEventIdx := len(redelgationEvents) - 1
	expectedEvent := &apptypes.EventRedelegation{
		Application:      expectedApp,
		SessionEndHeight: sessionEndHeight,
	}
	require.Equal(t, int(maxDelegatedGateways)+1, len(redelgationEvents))
	require.EqualValues(t, expectedEvent, redelgationEvents[lastEventIdx])

	// Verify that the application exists
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways-1, uint64(len(foundApp.DelegateeGatewayAddresses)))

	for i, gatewayAddr := range expectedGatewayAddresses {
		require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[i])
	}
}

func TestMsgServer_UndelegateFromGateway_FailNotDelegated(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr1 := sample.AccAddress()
	gatewayAddr2 := sample.AccAddress()
	// Mock the gateway being staked via the staked gateway map
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr1, gwtypes.GatewayNotUnstaking)
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr2, gwtypes.GatewayNotUnstaking)

	// Prepare the application
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the undelegation message
	undelegateMsg := &apptypes.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
	}

	// Attempt to undelgate the application from the gateway
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.ErrorContains(t, err, apptypes.ErrAppNotDelegated.Error())
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	redelegationEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 0, len(redelegationEvents))

	// Prepare a delegation message
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	currentHeight := sdkCtx.BlockHeight()
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	expectedApp := &apptypes.Application{
		Address:                   stakeMsg.GetAddress(),
		Stake:                     stakeMsg.GetStake(),
		ServiceConfigs:            stakeMsg.GetServices(),
		DelegateeGatewayAddresses: []string{gatewayAddr2},
		PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
	}
	expectedRedelegationEvent := &apptypes.EventRedelegation{
		Application:      expectedApp,
		SessionEndHeight: sessionEndHeight,
	}

	events = sdkCtx.EventManager().Events()
	redelegationEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 1, len(redelegationEvents))
	require.EqualValues(t, expectedRedelegationEvent, redelegationEvents[0])

	// Reset the events, as if a new block were created.
	ctx, sdkCtx = testevents.ResetEventManager(ctx)

	// Ensure the failed undelegation did not affect the application
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.ErrorContains(t, err, apptypes.ErrAppNotDelegated.Error())

	events = sdkCtx.EventManager().Events()
	require.Equal(t, 0, len(events), "expected no events")

	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr2, foundApp.DelegateeGatewayAddresses[0])
}

func TestMsgServer_UndelegateFromGateway_SuccessfullyUndelegateFromUnstakedGateway(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Generate an address for the application and gateways
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	// Mock the gateway being staked via the staked gateway map
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr, gwtypes.GatewayNotUnstaking)

	// Prepare the application
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message and delegate the application to the gateway
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
	expectedRedelegationEvent := &apptypes.EventRedelegation{
		Application:      expectedApp,
		SessionEndHeight: sessionEndHeight,
	}

	events := sdkCtx.EventManager().Events()
	redelegationEvents := testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 1, len(redelegationEvents))
	require.EqualValues(t, expectedRedelegationEvent, redelegationEvents[0])

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayAddresses))
	require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[0])

	// Reset the events, as if a new block were created.
	ctx, sdkCtx = testevents.ResetEventManager(ctx)

	// Mock unstaking the gateway
	keepertest.RemoveGatewayFromStakedGatewayMap(t, gatewayAddr)

	// Prepare an undelegation message
	undelegateMsg := &apptypes.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Undelegate the application from the gateway
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.NoError(t, err)

	events = sdkCtx.EventManager().Events()
	redelegationEvents = testevents.FilterEvents[*apptypes.EventRedelegation](t, events)
	require.Equal(t, 1, len(redelegationEvents))

	expectedApp = &apptypes.Application{
		Address:                   stakeMsg.GetAddress(),
		Stake:                     stakeMsg.GetStake(),
		ServiceConfigs:            stakeMsg.GetServices(),
		DelegateeGatewayAddresses: make([]string, 0),
		PendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
			uint64(sessionEndHeight): {GatewayAddresses: []string{undelegateMsg.GetGatewayAddress()}},
		},
	}
	expectedEvent := &apptypes.EventRedelegation{
		Application:      expectedApp,
		SessionEndHeight: sessionEndHeight,
	}
	for _, event := range redelegationEvents {
		require.EqualValues(t, expectedEvent, event)
	}

	// Verify that the application exists
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))
}

// Test an undelegation at different stages of the undelegation lifecycle:
//
//   - Create an application, stake it, delegate then undelegate it from a gateway.
//
//   - Increment the block height without moving to the next session and check that
//     the undelegated gateway is still part of the application's delegate gateways.
//
//   - Increment the block height to the next session and check that the undelegated
//     gateway is no longer part of the application's delegate gateways.
//
//   - Increment the block height past the tested session's grace period and check:
//
//   - The undelegated gateway is still not part of the application's delegate gateways
//
//   - If queried for a past block height, corresponding to the session at which the
//     undelegation occurred, the reconstructed delegate gateway list does include
//     the undelegated gateway.
func TestMsgServer_UndelegateFromGateway_DelegationIsActiveUntilNextSession(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	undelegationHeight := int64(1)
	sdkCtx, app, delegateAddr, pendingUndelegateFromAddr :=
		createAppStakeDelegateAndUndelegate(ctx, t, srv, k, undelegationHeight)

	// Increment the block height without moving to the next session, then run the
	// pruning undelegations logic.
	sdkCtx = sdkCtx.WithBlockHeight(undelegationHeight + 1)
	k.EndBlockerPruneAppToGatewayPendingUndelegation(sdkCtx)

	// Get the updated application state after pruning logic is run.
	app, isAppFound := k.GetApplication(sdkCtx, app.Address)
	require.True(t, isAppFound)
	require.NotNil(t, app)

	// Verify that the gateway was removed from the application's delegatee gateway addresses.
	require.NotContains(t, app.DelegateeGatewayAddresses, pendingUndelegateFromAddr)

	// Verify that the gateway is added to the pending undelegation list with the
	// right sessionEndHeight as the map key.
	sessionEndHeight := testsession.GetSessionEndHeightWithDefaultParams(undelegationHeight)
	require.Contains(t,
		app.PendingUndelegations[uint64(sessionEndHeight)].GatewayAddresses,
		pendingUndelegateFromAddr,
	)

	// Verify that the application is still delegating to other gateways.
	require.Contains(t, app.DelegateeGatewayAddresses, delegateAddr)

	// Verify that the reconstructed delegatee gateway list includes the undelegated gateway.
	gatewayAddresses := getRingAddressesAtBlockWithDefaultParams(&app, sdkCtx.BlockHeight())
	require.Contains(t, gatewayAddresses, pendingUndelegateFromAddr)

	// Increment the block height to the next session and run the pruning
	// undelegations logic again.
	nextSessionStartHeight := sessionEndHeight + 1
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)
	k.EndBlockerPruneAppToGatewayPendingUndelegation(sdkCtx)

	// Get the updated application state.
	app, isAppFound = k.GetApplication(sdkCtx, app.Address)
	require.True(t, isAppFound)
	require.NotNil(t, app)

	// Verify that when queried for the next session the reconstructed delegatee
	// gateway list does not include the undelegated gateway.
	nextSessionGatewayAddresses := getRingAddressesAtBlockWithDefaultParams(&app, nextSessionStartHeight)
	require.NotContains(t, nextSessionGatewayAddresses, pendingUndelegateFromAddr)

	// Increment the block height past the tested session's grace period and run
	// the pruning undelegations logic again.
	sharedParams := sharedtypes.DefaultParams()
	afterSessionGracePeriodEndHeight := sharedtypes.GetSessionGracePeriodEndHeight(&sharedParams, sessionEndHeight) + 1
	sdkCtx = sdkCtx.WithBlockHeight(afterSessionGracePeriodEndHeight)
	k.EndBlockerPruneAppToGatewayPendingUndelegation(sdkCtx)

	// Verify that when queried for a block height past the tested session's grace period,
	// the reconstructed delegatee gateway list does not include the undelegated gateway.
	pastGracePeriodGatewayAddresses := getRingAddressesAtBlockWithDefaultParams(&app, afterSessionGracePeriodEndHeight)
	require.NotContains(t, pastGracePeriodGatewayAddresses, pendingUndelegateFromAddr)

	// Ensure that when queried for the block height corresponding to the session
	// at which the undelegation occurred, the reconstructed delegatee gateway list
	// includes the undelegated gateway.
	gatewayAddressesBeforeUndelegation := getRingAddressesAtBlockWithDefaultParams(&app, int64(sessionEndHeight))
	require.Contains(t, gatewayAddressesBeforeUndelegation, pendingUndelegateFromAddr)
}

func TestMsgServer_UndelegateFromGateway_DelegationIsPrunedAfterRetentionPeriod(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	undelegationHeight := int64(1)
	sdkCtx, app, delegateAddr, pendingUndelegateFromAddr :=
		createAppStakeDelegateAndUndelegate(ctx, t, srv, k, undelegationHeight)

	// Increment the block height past the undelegation retention period then run
	// the pruning undelegations logic.
	pruningBlockHeight := getUndelegationPruningBlockHeight(undelegationHeight)
	sdkCtx = sdkCtx.WithBlockHeight(pruningBlockHeight)
	k.EndBlockerPruneAppToGatewayPendingUndelegation(sdkCtx)

	// Get the updated application state.
	app, isAppFound := k.GetApplication(sdkCtx, app.Address)
	require.True(t, isAppFound)
	require.NotNil(t, app)

	// Verify that the pending undelegation map no longer contains the
	// sessionEndHeight key.
	sessionEndHeight := uint64(testsession.GetSessionEndHeightWithDefaultParams(undelegationHeight))
	require.Empty(t, app.PendingUndelegations[sessionEndHeight])

	// Verify that the reconstructed delegatee gateway list can no longer include
	// the undelegated gateway since it has been pruned.
	gatewayAddressesAfterPruning := getRingAddressesAtBlockWithDefaultParams(&app, undelegationHeight)
	require.NotContains(t, gatewayAddressesAfterPruning, pendingUndelegateFromAddr)
	require.Contains(t, gatewayAddressesAfterPruning, delegateAddr)
}

func TestMsgServer_UndelegateFromGateway_RedelegationAfterUndelegationAtTheSameSessionNumber(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	undelegationHeight := int64(1)
	sdkCtx, app, _, gatewayAddrToRedelegate :=
		createAppStakeDelegateAndUndelegate(ctx, t, srv, k, undelegationHeight)

	// Increment the block height without moving to the next session.
	sdkCtx = sdkCtx.WithBlockHeight(undelegationHeight + 1)

	// Delegate back the application to the gateway that was undelegated from.
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     app.Address,
		GatewayAddress: gatewayAddrToRedelegate,
	}
	_, err := srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	// Get the updated application state.
	app, isAppFound := k.GetApplication(sdkCtx, app.Address)
	require.True(t, isAppFound)
	require.NotNil(t, app)

	// Verify that the gateway is still in the application's delegatee gateway addresses.
	require.Contains(t, app.DelegateeGatewayAddresses, gatewayAddrToRedelegate)

	// Verify that the gateway is also present in the pending undelegation list with the
	// right sessionEndHeight as the map key.
	sessionEndHeight := uint64(testsession.GetSessionEndHeightWithDefaultParams(undelegationHeight))
	require.NotContains(t,
		app.PendingUndelegations[sessionEndHeight].GatewayAddresses,
		gatewayAddrToRedelegate,
	)

	// Verify that the reconstructed delegatee gateway list includes the redelegated gateway.
	gatewayAddresses := getRingAddressesAtBlockWithDefaultParams(&app, sdkCtx.BlockHeight())
	require.Contains(t, gatewayAddresses, gatewayAddrToRedelegate)

	// Increment the block height past the undelegation retention period then run
	// the pruning undelegations logic.
	pruningBlockHeight := getUndelegationPruningBlockHeight(undelegationHeight)
	sdkCtx = sdkCtx.WithBlockHeight(pruningBlockHeight)
	k.EndBlockerPruneAppToGatewayPendingUndelegation(sdkCtx)

	// Get the updated application state after pruning logic is run.
	app, isAppFound = k.GetApplication(sdkCtx, app.Address)
	require.True(t, isAppFound)
	require.NotNil(t, app)

	// Verify that the application is still delegated to the gateway
	require.Contains(t, app.DelegateeGatewayAddresses, gatewayAddrToRedelegate)

	// Verify that the pending undelegation map no longer contains the
	// sessionEndHeight key.
	require.Empty(t, app.PendingUndelegations[sessionEndHeight])

	// Verify that the reconstructed delegatee gateway list includes the redelegated gateway
	gatewayAddressesAfterPruning := getRingAddressesAtBlockWithDefaultParams(&app, sdkCtx.BlockHeight())
	require.Contains(t, gatewayAddressesAfterPruning, gatewayAddrToRedelegate)
}

func TestMsgServer_UndelegateFromGateway_UndelegateFromUnbondingGateway(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	undelegationHeight := int64(1)
	sdkCtx, app, delegateAddr, _ :=
		createAppStakeDelegateAndUndelegate(ctx, t, srv, k, undelegationHeight)

	require.Contains(t, app.DelegateeGatewayAddresses, delegateAddr)

	sessionEndHeight := testsession.GetSessionEndHeightWithDefaultParams(undelegationHeight)
	// Increment the block height to make the gateway inactive.
	sdkCtx = sdkCtx.WithBlockHeight(sessionEndHeight + 1)

	err := k.EndBlockerAutoUndelegateFromUnbondingGateways(sdkCtx)
	require.NoError(t, err)

	app, isAppFound := k.GetApplication(sdkCtx, app.Address)
	require.True(t, isAppFound)
	require.Len(t, app.DelegateeGatewayAddresses, 0)
	require.Len(t, app.PendingUndelegations[uint64(sessionEndHeight)].GatewayAddresses, 1)
}

// createAppStakeDelegateAndUndelegate is a helper function that is used in the tests
// that exercise the pruning undelegations and ring addresses reconstruction logic.
// * It creates an account address and stakes it as an application.
// * Creates two gateway addresses and mocks them being staked.
// * Delegates the application to the gateways.
// * Undelegates the application from one of the gateways.
func createAppStakeDelegateAndUndelegate(
	ctx context.Context,
	t *testing.T,
	srv apptypes.MsgServer,
	k keeper.Keeper,
	undelegationHeight int64,
) (
	sdkCtx sdk.Context,
	app apptypes.Application,
	delegateAddr,
	pendingUndelegateFromAddr string,
) {
	// Generate an application address and stake the application.
	appAddr := sample.AccAddress()
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &apptypes.DefaultMinStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "svc1"},
		},
	}
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	sessionEndHeight := uint64(testsession.GetSessionEndHeightWithDefaultParams(undelegationHeight))

	// Generate gateway addresses, mock the gateways being staked then delegate the
	// application to the gateways.
	delegateAddr = sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, delegateAddr, sessionEndHeight)

	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: delegateAddr,
	}
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	pendingUndelegateFromAddr = sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, pendingUndelegateFromAddr, sessionEndHeight)

	delegateMsg = &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: pendingUndelegateFromAddr,
	}
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	// Create a context with a block height of 2.
	sdkCtx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(undelegationHeight)

	// Undelegate from the first gateway.
	undelegateMsg := &apptypes.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: pendingUndelegateFromAddr,
	}
	_, err = srv.UndelegateFromGateway(sdkCtx, undelegateMsg)
	require.NoError(t, err)

	foundApp, isAppFound := k.GetApplication(sdkCtx, appAddr)

	// Verify that the application exists.
	require.True(t, isAppFound)
	require.NotNil(t, foundApp)

	return sdkCtx, foundApp, delegateAddr, pendingUndelegateFromAddr
}

// getUndelegationPruningBlockHeight returns the block height at which undelegations
// should be pruned for a given undlegation block height.
func getUndelegationPruningBlockHeight(blockHeight int64) (pruningHeihgt int64) {
	nextSessionStartHeight := testsession.GetSessionEndHeightWithDefaultParams(blockHeight) + 1

	return nextSessionStartHeight + getNumBlocksUndelegationRetentionWithDefaultParams()
}

// getNumBlocksUndelegationRetentionWithDefaultParams returns the number of blocks for
// which undelegations should be kept before being pruned, given the default shared
// module parameters.
func getNumBlocksUndelegationRetentionWithDefaultParams() int64 {
	sharedParams := sharedtypes.DefaultParams()
	return keeper.GetNumBlocksUndelegationRetention(&sharedParams)
}

// getRingAddressesAtBlockWithDefaultParams returns the active gateway addresses that
// need to be used to construct the ring in order to validate that the given app should
// pay for.
// It takes into account both active delegations and pending undelegations that
// should still be part of the ring at the given block height.
// The ring addresses slice is reconstructed by adding back the past delegated
// gateways that have been undelegated after the target session end height.
func getRingAddressesAtBlockWithDefaultParams(app *apptypes.Application, blockHeight int64) []string {
	sharedParams := sharedtypes.DefaultParams()
	return rings.GetRingAddressesAtBlock(&sharedParams, app, blockHeight)
}
