package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
)

func TestMsgServer_DelegateToGateway_SuccessfullyDelegate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateways
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

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayPubKeys))
	foundPubKey, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[0])
	require.NoError(t, err)
	foundGatewayAddr := types.PublicKeyToAddress(foundPubKey)
	require.Equal(t, gatewayAddr1, foundGatewayAddr)

	// Prepare a second delegation message
	delegateMsg2 := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr2,
	}

	// Delegate the application to the second gateway
	_, err = srv.DelegateToGateway(wctx, delegateMsg2)
	require.NoError(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 2, len(foundApp.DelegateeGatewayPubKeys))
	foundPubKey1, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[0])
	require.NoError(t, err)
	foundGatewayAddr1 := types.PublicKeyToAddress(foundPubKey1)
	require.Equal(t, gatewayAddr1, foundGatewayAddr1)
	foundPubKey2, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[1])
	require.NoError(t, err)
	foundGatewayAddr2 := types.PublicKeyToAddress(foundPubKey2)
	require.Equal(t, gatewayAddr2, foundGatewayAddr2)
}

func TestMsgServer_DelegateToGateway_FailDuplicate(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)

	// Generate an address for the application and gateway
	appAddr := sample.AccAddress()
	gatewayAddr, gatewayPubKey := sample.AddrAndPubKey()
	keepertest.AddrToPubKeyMap[gatewayAddr] = gatewayPubKey
	defer delete(keepertest.AddrToPubKeyMap, gatewayAddr)

	// Prepare the application
	stakeMsg := &types.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
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

	// Verify that the application exists
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, appAddr, foundApp.Address)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayPubKeys))
	foundPubKey, err := types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[0])
	require.NoError(t, err)
	foundGatewayAddr := types.PublicKeyToAddress(foundPubKey)
	require.Equal(t, gatewayAddr, foundGatewayAddr)

	// Prepare a second delegation message
	delegateMsg2 := &types.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Attempt to delegate the application to the gateway again
	_, err = srv.DelegateToGateway(wctx, delegateMsg2)
	require.Error(t, err)
	foundApp, isAppFound = k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 1, len(foundApp.DelegateeGatewayPubKeys))
	foundPubKey, err = types.AnyToPubKey(foundApp.DelegateeGatewayPubKeys[0])
	require.NoError(t, err)
	foundGatewayAddr = types.PublicKeyToAddress(foundPubKey)
	require.Equal(t, gatewayAddr, foundGatewayAddr)
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
	require.Error(t, err)
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	require.True(t, isAppFound)
	require.Equal(t, 0, len(foundApp.DelegateeGatewayPubKeys))
}
