package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_UndelegateFromGateway_SuccessfullyUndelegate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

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
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
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
		_, err = srv.DelegateToGateway(ctx, delegateMsg)
		require.NoError(t, err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	require.Equal(t, int(maxDelegatedGateways), len(events))

	for i, event := range events {
		require.Equal(t, "poktroll.application.EventRedelegation", event.Type)
		require.Equal(t, "app_address", event.Attributes[0].Key)
		require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), event.Attributes[0].Value)
		require.Equal(t, "gateway_address", event.Attributes[1].Key)
		require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddresses[i]), event.Attributes[1].Value)
	}

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways, uint64(len(foundApp.DelegateeGatewayAddresses)))

	for i, gatewayAddr := range gatewayAddresses {
		require.Equal(t, gatewayAddr, foundApp.DelegateeGatewayAddresses[i])
	}

	// Prepare an undelegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddresses[3],
	}

	// Undelegate the application from the gateway
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.NoError(t, err)

	events = sdkCtx.EventManager().Events()
	require.Equal(t, int(maxDelegatedGateways)+1, len(events))
	require.Equal(t, "poktroll.application.EventRedelegation", events[7].Type)
	require.Equal(t, "app_address", events[7].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[7].Attributes[0].Value)
	require.Equal(t, "gateway_address", events[7].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddresses[3]), events[7].Attributes[1].Value)

	// Verify that the application exists
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways-1, uint64(len(foundApp.DelegateeGatewayAddresses)))

	gatewayAddresses = append(gatewayAddresses[:3], gatewayAddresses[4:]...)
	for i, gatewayAddr := range gatewayAddresses {
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
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr1)
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr2)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the undelegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr1,
	}

	// Attempt to undelgate the application from the gateway
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.ErrorIs(t, err, types.ErrAppNotDelegated)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayAddresses))

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	events := sdkCtx.EventManager().Events()
	require.Equal(t, 0, len(events))

	// Prepare a delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	events = sdkCtx.EventManager().Events()
	require.Equal(t, 1, len(events))
	require.Equal(t, "poktroll.application.EventRedelegation", events[0].Type)
	require.Equal(t, "app_address", events[0].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[0].Attributes[0].Value)
	require.Equal(t, "gateway_address", events[0].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddr2), events[0].Attributes[1].Value)

	// Ensure the failed undelegation did not affect the application
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.ErrorIs(t, err, types.ErrAppNotDelegated)

	events = sdkCtx.EventManager().Events()
	require.Equal(t, 1, len(events))

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
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message and delegate the application to the gateway
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
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[0].Attributes[0].Value)
	require.Equal(t, "gateway_address", events[0].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddr), events[0].Attributes[1].Value)

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
	_, err = srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.NoError(t, err)

	events = sdkCtx.EventManager().Events()
	require.Equal(t, 2, len(events))
	require.Equal(t, "poktroll.application.EventRedelegation", events[1].Type)
	require.Equal(t, "app_address", events[1].Attributes[0].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", appAddr), events[1].Attributes[0].Value)
	require.Equal(t, "gateway_address", events[0].Attributes[1].Key)
	require.Equal(t, fmt.Sprintf("\"%s\"", gatewayAddr), events[0].Attributes[1].Value)

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
	afterSessionGracePeriodEndHeight := shared.GetSessionGracePeriodEndHeight(&sharedParams, sessionEndHeight) + 1
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

	// Verify that the the pending undelegation map no longer contains the
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
	delegateMsg := &types.MsgDelegateToGateway{
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
	require.Contains(t,
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

	// Verify that the the pending undelegation map no longer contains the
	// sessionEndHeight key.
	require.Empty(t, app.PendingUndelegations[sessionEndHeight])

	// Verify that the reconstructed delegatee gateway list includes the redelegated gateway
	gatewayAddressesAfterPruning := getRingAddressesAtBlockWithDefaultParams(&app, sdkCtx.BlockHeight())
	require.Contains(t, gatewayAddressesAfterPruning, gatewayAddrToRedelegate)
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
	srv types.MsgServer,
	k keeper.Keeper,
	undelegationHeight int64,
) (
	sdkCtx sdk.Context,
	app types.Application,
	delegateAddr,
	pendingUndelegateFromAddr string,
) {
	// Generate an application address and stake the application.
	appAddr := sample.AccAddress()
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Generate gateway addresses, mock the gateways being staked then delegate the
	// application to the gateways.
	delegateAddr = sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, delegateAddr)

	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: delegateAddr,
	}
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	pendingUndelegateFromAddr = sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, pendingUndelegateFromAddr)

	delegateMsg = &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: pendingUndelegateFromAddr,
	}
	_, err = srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)

	// Create a context with a block height of 2.
	sdkCtx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(undelegationHeight)

	// Undelegate from the first gateway.
	undelegateMsg := &types.MsgUndelegateFromGateway{
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
