package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
	sharedtypes "pocket/x/shared/types"
)

func TestMsgServer_UndelegateFromGateway_SuccessfullyUndelegate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateways
	appAddr := sample.AccAddress()
	gatewayAddresses := make([]string, int(k.GetParams(ctx).MaxDelegatedGateways))
	for i := 0; i < len(gatewayAddresses); i++ {
		gatewayAddr, gatewayPubKey := sample.AddrAndPubKey()
		keepertest.AddrToPubKeyMap[gatewayAddr] = gatewayPubKey
		defer delete(keepertest.AddrToPubKeyMap, gatewayAddr)
		gatewayAddresses[i] = gatewayAddr
	}

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
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

	// Verify that the application exists
	maxDelegatedGateways := k.GetParams(ctx).MaxDelegatedGateways
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways, int64(len(foundApp.DelegateeGatewayPubKeys)))
	for i, gatewayAddr := range gatewayAddresses {
		foundPubKey, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[i])
		require.NoError(t, err)
		foundGatewayAddr := types.PublicKeyToAddress(foundPubKey)
		require.Equal(t, gatewayAddr, foundGatewayAddr)
	}

	// Prepare an undelegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddresses[3],
	}

	// Undelegate the application from the gateway
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, maxDelegatedGateways-1, int64(len(foundApp.DelegateeGatewayPubKeys)))
	gatewayAddresses = append(gatewayAddresses[:3], gatewayAddresses[4:]...)
	for i, gatewayAddr := range gatewayAddresses {
		foundPubKey, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[i])
		require.NoError(t, err)
		foundGatewayAddr := types.PublicKeyToAddress(foundPubKey)
		require.Equal(t, gatewayAddr, foundGatewayAddr)
	}
}

func TestMsgServer_UndelegateFromGateway_FailNotDelegated(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr1, gatewayPubKey1 := sample.AddrAndPubKey()
	gatewayAddr2, gatewayPubKey2 := sample.AddrAndPubKey()
	keepertest.AddrToPubKeyMap[gatewayAddr1] = gatewayPubKey1
	keepertest.AddrToPubKeyMap[gatewayAddr2] = gatewayPubKey2
	defer delete(keepertest.AddrToPubKeyMap, gatewayAddr1)
	defer delete(keepertest.AddrToPubKeyMap, gatewayAddr2)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
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
	require.Error(t, err)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayPubKeys))

	// Prepare a delegation message
	delegateMsg := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the gateway
	_, err = srv.DelegateToGateway(wctx, delegateMsg)
	require.NoError(t, err)

	// Ensure the failed undelegation did not affect the application
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.Error(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayPubKeys))
	foundPubKey, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[0])
	require.NoError(t, err)
	foundGatewayAddr := types.PublicKeyToAddress(foundPubKey)
	require.Equal(t, gatewayAddr2, foundGatewayAddr)
}

// TODO_TECHDEBT(@h5law): This should not be a cause of failure for unstaking
func TestMsgServer_UndelegateFromGateway_FailGatewayNotStaked(t *testing.T) {
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
				ServiceId: &sharedtypes.ServiceId{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(wctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)

	// Prepare the delegation message
	undelegateMsg := &types.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the unstaked gateway
	_, err = srv.UndelegateFromGateway(wctx, undelegateMsg)
	require.Error(t, err)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayPubKeys))
}
