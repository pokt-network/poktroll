package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_TransferApplication_Success(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)
	sharedParams := sharedtypes.DefaultParams()

	// Generate an address for the source and destination applications.
	srcBech32 := sample.AccAddress()
	dstBech32 := sample.AccAddress()

	// Verify that the app does not exist yet.
	_, isSrcFound := k.GetApplication(ctx, srcBech32)
	require.False(t, isSrcFound)

	expectedAppStake := &apptypes.DefaultMinStake

	// Prepare the application.
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: srcBech32,
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
	srcApp, isSrcFound := k.GetApplication(ctx, srcBech32)
	require.True(t, isSrcFound)
	require.Equal(t, srcBech32, srcApp.GetAddress())
	require.Equal(t, expectedAppStake, srcApp.GetStake())
	require.Len(t, srcApp.GetServiceConfigs(), 1)
	require.Equal(t, "svc1", srcApp.GetServiceConfigs()[0].GetServiceId())

	transferBeginHeight := cosmostypes.UnwrapSDKContext(ctx).BlockHeight()
	transferBeginSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferBeginHeight)
	expectedPendingTransfer := &apptypes.PendingApplicationTransfer{
		DestinationAddress: dstBech32,
		SessionEndHeight:   uint64(transferBeginSessionEndHeight),
	}

	// Transfer the application stake from the source to the destination application address.
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	transferStakeMsg := apptypes.NewMsgTransferApplication(srcBech32, dstBech32)
	_, stakeTransferErr := srv.TransferApplication(ctx, transferStakeMsg)
	require.NoError(t, stakeTransferErr)
	getTransferHeightApp := &apptypes.Application{
		Address: srcBech32,
		PendingTransfer: &apptypes.PendingApplicationTransfer{
			DestinationAddress: dstBech32,
			SessionEndHeight:   uint64(sessionEndHeight),
		},
	}

	// Assert that the EventTransferBegin event was emitted.
	expectedApp := srcApp
	expectedApp.PendingTransfer = &apptypes.PendingApplicationTransfer{
		DestinationAddress: dstBech32,
		SessionEndHeight:   uint64(transferBeginSessionEndHeight),
	}
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, getTransferHeightApp)
	expectedTransferBeginEvent := &apptypes.EventTransferBegin{
		SourceAddress:      srcBech32,
		DestinationAddress: dstBech32,
		SourceApplication:  &expectedApp,
		SessionEndHeight:   transferBeginSessionEndHeight,
		TransferEndHeight:  transferEndHeight,
	}
	events := cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()
	transferBeginEvents := testutilevents.FilterEvents[*apptypes.EventTransferBegin](t, events)
	require.Equal(t, 1, len(transferBeginEvents), "expected 1 transfer begin event")
	require.EqualValues(t, expectedTransferBeginEvent, transferBeginEvents[0])

	// Set the height to the transfer end height - 1 for the session.
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithBlockHeight(transferEndHeight - 1)

	// Run application module end-blockers to assert that the transfer is not completed yet.
	err = k.EndBlockerTransferApplication(ctx)
	require.NoError(t, err)

	// Assert that the source app is still pending transfer.
	pendingSrcApp, isSrcFound := k.GetApplication(ctx, srcBech32)
	require.True(t, isSrcFound)
	require.EqualValues(t, expectedPendingTransfer, pendingSrcApp.GetPendingTransfer())

	// Assert that the destination app was not created yet.
	_, isDstFound := k.GetApplication(ctx, dstBech32)
	require.False(t, isDstFound)

	// Set the height to the transfer end height for the session.
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithBlockHeight(transferEndHeight)

	// Run application module end-blockers to complete the transfer.
	err = k.EndBlockerTransferApplication(ctx)
	require.NoError(t, err)

	// Verify that the destination app was created with the correct state.
	dstApp, isDstFound := k.GetApplication(ctx, dstBech32)
	require.True(t, isDstFound)
	require.Equal(t, dstBech32, dstApp.GetAddress())
	require.Equal(t, expectedAppStake, dstApp.GetStake())
	require.Len(t, dstApp.GetServiceConfigs(), 1)
	require.Equal(t, "svc1", dstApp.GetServiceConfigs()[0].GetServiceId())

	// Assert that the EventTransferEnd event was emitted.
	transferEndSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferEndHeight)
	expectedTransferEndEvent := &apptypes.EventTransferEnd{
		SourceAddress:          srcBech32,
		DestinationAddress:     dstBech32,
		DestinationApplication: &dstApp,
		SessionEndHeight:       transferEndSessionEndHeight,
		TransferEndHeight:      transferEndHeight,
	}
	events = cosmostypes.UnwrapSDKContext(ctx).EventManager().Events()
	transferEndEvents := testutilevents.FilterEvents[*apptypes.EventTransferEnd](t, events)
	require.Equal(t, 1, len(transferEndEvents), "expected 1 transfer end event")
	require.EqualValues(t, expectedTransferEndEvent, transferEndEvents[0])

	srcApp.Address = ""
	dstApp.Address = ""
	require.EqualValues(t, srcApp, dstApp)

	// Verify that the source app was unstaked.
	srcApp, isSrcFound = k.GetApplication(ctx, srcBech32)
	require.False(t, isSrcFound)
}
