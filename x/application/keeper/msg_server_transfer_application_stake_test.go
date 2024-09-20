package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_TransferApplication_Success(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)

	// Generate an address for the source and destination applications.
	srcAddr := sample.AccAddress()
	dstAddr := sample.AccAddress()

	// Verify that the app does not exist yet.
	_, isSrcFound := k.GetApplication(ctx, srcAddr)
	require.False(t, isSrcFound)

	expectedAppStake := &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(100)}

	// Prepare the application.
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: srcAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	// Stake the application.
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)

	// Verify that the application exists.
	srcApp, isSrcFound := k.GetApplication(ctx, srcAddr)
	require.True(t, isSrcFound)
	require.Equal(t, srcAddr, srcApp.GetAddress())
	require.Equal(t, expectedAppStake, srcApp.GetStake())
	require.Len(t, srcApp.GetServiceConfigs(), 1)
	require.Equal(t, "svc1", srcApp.GetServiceConfigs()[0].GetServiceId())

	// Transfer the application stake from the source to the destination application address.
	transferStakeMsg := apptypes.NewMsgTransferApplication(srcAddr, dstAddr)

	transferAppStakeRes, stakeTransferErr := srv.TransferApplication(ctx, transferStakeMsg)
	require.NoError(t, stakeTransferErr)
	transferResApp := transferAppStakeRes.GetApplication()
	require.NotNil(t, transferResApp.GetPendingTransfer())

	// Assert that the source app and the transfer response app are the same except for the #PendingTransfer.
	transferResAppCopy := *transferResApp
	transferResAppCopy.PendingTransfer = nil
	require.EqualValues(t, srcApp, transferResAppCopy)

	// Set the height to the proof window close height for the session.
	sharedParams := sharedtypes.DefaultParams()
	proofWindowCloseHeight := apptypes.GetApplicationTransferHeight(&sharedParams, transferResApp)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithBlockHeight(proofWindowCloseHeight)

	// Run application module end-blockers to complete the transfer.
	err = k.EndBlockerTransferApplication(ctx)
	require.NoError(t, err)

	// Verify that the destination app was created with the correct state.
	dstApp, isDstFound := k.GetApplication(ctx, dstAddr)
	require.True(t, isDstFound)
	require.Equal(t, dstAddr, dstApp.GetAddress())
	require.Equal(t, expectedAppStake, dstApp.GetStake())
	require.Len(t, dstApp.GetServiceConfigs(), 1)
	require.Equal(t, "svc1", dstApp.GetServiceConfigs()[0].GetServiceId)

	srcApp.Address = ""
	dstApp.Address = ""
	require.EqualValues(t, srcApp, dstApp)

	// Verify that the source app was unstaked.
	srcApp, isSrcFound = k.GetApplication(ctx, srcAddr)
	require.False(t, isSrcFound)
}

func TestMsgServer_TransferApplication_Error_DestinationExists(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)

	// Generate an address for the source and destination applications.
	srcAddr := sample.AccAddress()
	dstAddr := sample.AccAddress()

	// Verify that neither the source nor the destination application exists yet.
	_, isSrcFound := k.GetApplication(ctx, srcAddr)
	require.False(t, isSrcFound)

	_, isDstFound := k.GetApplication(ctx, dstAddr)
	require.False(t, isDstFound)

	expectedAppStake := &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(100)}

	// Prepare and stake the application.
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address: srcAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	_, err := srv.StakeApplication(ctx, appStakeMsg)
	require.NoError(t, err)

	// Prepare and stake the destination application.
	dstAppStakeMsg := &apptypes.MsgStakeApplication{
		Address: dstAddr,
		Stake:   expectedAppStake,
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: "svc1",
			},
		},
	}

	_, err = srv.StakeApplication(ctx, dstAppStakeMsg)
	require.NoError(t, err)

	// Attempt to transfer the source application stake to the destination.
	transferStakeMsg := apptypes.NewMsgTransferApplication(srcAddr, dstAddr)

	_, err = srv.TransferApplication(ctx, transferStakeMsg)
	require.ErrorContains(t, err, apptypes.ErrAppDuplicateAddress.Wrapf("destination application (%s) exists", dstAddr).Error())

	// Verify that the original application still exists.
	var foundApp apptypes.Application
	foundApp, isSrcFound = k.GetApplication(ctx, srcAddr)
	require.True(t, isSrcFound)
	require.Equal(t, srcAddr, foundApp.Address)
	require.Equal(t, int64(100), foundApp.Stake.Amount.Int64())
	require.Len(t, foundApp.ServiceConfigs, 1)
	require.Equal(t, "svc1", foundApp.ServiceConfigs[0].GetServiceId())
}
