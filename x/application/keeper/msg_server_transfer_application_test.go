package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	pocket "github.com/pokt-network/poktroll/app/pocket"
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
	srcBech32 := sample.AccAddressBech32()
	dstBech32 := sample.AccAddressBech32()

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

	// Set the height to the next session end height at or after the transfer end height.
	// The transfer can only complete at a session end height.
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx)
	transferCompletionHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferEndHeight)
	ctx = sdkCtx.WithBlockHeight(transferCompletionHeight)

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
	transferEndSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferCompletionHeight)
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

func TestMsgServer_TransferApplication_MergePerSessionSpendLimit(t *testing.T) {
	fivePOKT := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(5000000))
	threePOKT := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(3000000))

	tests := []struct {
		desc          string
		srcSpendLimit *cosmostypes.Coin
		dstSpendLimit *cosmostypes.Coin
		expectedLimit *cosmostypes.Coin
	}{
		{
			desc:          "both apps have nil spend limit",
			srcSpendLimit: nil,
			dstSpendLimit: nil,
			expectedLimit: nil,
		},
		{
			desc:          "only source has spend limit",
			srcSpendLimit: &fivePOKT,
			dstSpendLimit: nil,
			expectedLimit: &fivePOKT,
		},
		{
			desc:          "only destination has spend limit",
			srcSpendLimit: nil,
			dstSpendLimit: &fivePOKT,
			expectedLimit: &fivePOKT,
		},
		{
			desc:          "both have spend limits - source is lower",
			srcSpendLimit: &threePOKT,
			dstSpendLimit: &fivePOKT,
			expectedLimit: &threePOKT,
		},
		{
			desc:          "both have spend limits - destination is lower",
			srcSpendLimit: &fivePOKT,
			dstSpendLimit: &threePOKT,
			expectedLimit: &threePOKT,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			k, ctx := keepertest.ApplicationKeeper(t)
			srv := appkeeper.NewMsgServerImpl(k)
			sharedParams := sharedtypes.DefaultParams()

			srcBech32 := sample.AccAddressBech32()
			dstBech32 := sample.AccAddressBech32()

			defaultStake := &apptypes.DefaultMinStake
			svcConfigs := []*sharedtypes.ApplicationServiceConfig{
				{ServiceId: "svc1"},
			}

			// Stake the source application with its spend limit.
			srcStakeMsg := &apptypes.MsgStakeApplication{
				Address:              srcBech32,
				Stake:                defaultStake,
				Services:             svcConfigs,
				PerSessionSpendLimit: test.srcSpendLimit,
			}
			_, err := srv.StakeApplication(ctx, srcStakeMsg)
			require.NoError(t, err)

			// Initiate the transfer from source to destination.
			// The destination must NOT exist yet for TransferApplication to succeed.
			transferMsg := apptypes.NewMsgTransferApplication(srcBech32, dstBech32)
			_, err = srv.TransferApplication(ctx, transferMsg)
			require.NoError(t, err)

			// Stake the destination application AFTER the transfer is initiated
			// but BEFORE the transfer completes. This causes the EndBlocker to
			// merge the source into the existing destination (the code path that
			// exercises mergeAppPerSessionSpendLimit).
			dstStakeMsg := &apptypes.MsgStakeApplication{
				Address:              dstBech32,
				Stake:                defaultStake,
				Services:             svcConfigs,
				PerSessionSpendLimit: test.dstSpendLimit,
			}
			_, err = srv.StakeApplication(ctx, dstStakeMsg)
			require.NoError(t, err)

			// Fast-forward to the transfer completion height (next session end
			// at or after the transfer end height).
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			srcApp, isSrcFound := k.GetApplication(ctx, srcBech32)
			require.True(t, isSrcFound)
			transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &srcApp)
			transferCompletionHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferEndHeight)
			ctx = sdkCtx.WithBlockHeight(transferCompletionHeight)

			// Run the end blocker to complete the transfer and trigger the merge.
			err = k.EndBlockerTransferApplication(ctx)
			require.NoError(t, err)

			// Verify the destination application exists and has the expected spend limit.
			dstApp, isDstFound := k.GetApplication(ctx, dstBech32)
			require.True(t, isDstFound)

			if test.expectedLimit == nil {
				require.Nil(t, dstApp.PerSessionSpendLimit,
					"expected nil PerSessionSpendLimit, got %v", dstApp.PerSessionSpendLimit)
			} else {
				require.NotNil(t, dstApp.PerSessionSpendLimit,
					"expected PerSessionSpendLimit %v, got nil", test.expectedLimit)
				require.True(t, dstApp.PerSessionSpendLimit.Equal(*test.expectedLimit),
					"expected PerSessionSpendLimit %v, got %v", test.expectedLimit, dstApp.PerSessionSpendLimit)
			}

			// Verify the source application was removed.
			_, isSrcFound = k.GetApplication(ctx, srcBech32)
			require.False(t, isSrcFound)
		})
	}
}

// TestMsgServer_TransferApplication_MergeRecordsServiceConfigHistory is a
// regression test for the transfer-merge service-config-history bug: when a
// transfer merges the source into an existing destination that already had a
// service swap (non-empty ServiceConfigHistory), the service IDs merged in from
// the source must be recorded in the destination's history. Otherwise session
// hydration (which reads history once non-empty) cannot resolve the merged-in
// service. See transfer_applications.go mergeAppServiceConfigs.
func TestMsgServer_TransferApplication_MergeRecordsServiceConfigHistory(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)
	sharedParams := sharedtypes.DefaultParams()

	srcBech32 := sample.AccAddressBech32()
	dstBech32 := sample.AccAddressBech32()
	defaultStake := &apptypes.DefaultMinStake

	// Stake the source application serving svc1, then initiate the transfer to
	// the (not-yet-existing) destination.
	_, err := srv.StakeApplication(ctx, &apptypes.MsgStakeApplication{
		Address:  srcBech32,
		Stake:    defaultStake,
		Services: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
	})
	require.NoError(t, err)

	_, err = srv.TransferApplication(ctx, apptypes.NewMsgTransferApplication(srcBech32, dstBech32))
	require.NoError(t, err)

	// Stake the destination serving svc2, then swap it to svc3. The swap gives
	// the destination a NON-EMPTY ServiceConfigHistory — the precondition that
	// triggered the bug (the merge then took the early no-op path).
	_, err = srv.StakeApplication(ctx, &apptypes.MsgStakeApplication{
		Address:  dstBech32,
		Stake:    defaultStake,
		Services: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc2"}},
	})
	require.NoError(t, err)
	// Re-stake must raise the stake amount, so bump it for the swap.
	higherStake := apptypes.DefaultMinStake.AddAmount(math.NewInt(1000000))
	_, err = srv.StakeApplication(ctx, &apptypes.MsgStakeApplication{
		Address:  dstBech32,
		Stake:    &higherStake,
		Services: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc3"}},
	})
	require.NoError(t, err)

	dstBeforeMerge, isDstFound := k.GetApplication(ctx, dstBech32)
	require.True(t, isDstFound)
	require.NotEmpty(t, dstBeforeMerge.ServiceConfigHistory, "test setup: dst must have non-empty history")

	// Fast-forward to the transfer completion height and run the EndBlocker,
	// which merges src (svc1) into the existing dst (svc3).
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srcApp, isSrcFound := k.GetApplication(ctx, srcBech32)
	require.True(t, isSrcFound)
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &srcApp)
	transferCompletionHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferEndHeight)
	ctx = sdkCtx.WithBlockHeight(transferCompletionHeight)

	require.NoError(t, k.EndBlockerTransferApplication(ctx))

	// The merged destination must serve BOTH the merged-in svc1 and its own svc3,
	// resolvable via service config history at a post-merge session height.
	dstApp, isDstFound := k.GetApplication(ctx, dstBech32)
	require.True(t, isDstFound)

	queryHeight := transferCompletionHeight + 2*int64(sharedParams.GetNumBlocksPerSession())
	activeServiceIDs := make(map[string]struct{})
	for _, svc := range dstApp.GetActiveServiceConfigs(queryHeight) {
		activeServiceIDs[svc.GetServiceId()] = struct{}{}
	}

	require.Contains(t, activeServiceIDs, "svc1", "merged-in source service must be recorded in dst history")
	require.Contains(t, activeServiceIDs, "svc3", "destination's own service must remain active")
}

// TestMsgServer_TransferApplication_NewAddressRewritesHistoryAppAddress is a
// regression test for the transfer-to-new-address service config history
// ApplicationAddress bug: when an application with non-empty
// ServiceConfigHistory transfers to a new address (no merge), the copied
// history entries' ApplicationAddress field is rewritten to the destination's
// new address. Without the rewrite, indexers and queries that filter history
// by app address would miss the dst app's pre-transfer entries.
func TestMsgServer_TransferApplication_NewAddressRewritesHistoryAppAddress(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(k)

	srcBech32 := sample.AccAddressBech32()
	dstBech32 := sample.AccAddressBech32()

	// Stake src serving svc1, then swap to svc2 to populate ServiceConfigHistory
	// with two entries (both naming srcBech32 as the ApplicationAddress).
	stake := apptypes.DefaultMinStake
	_, err := srv.StakeApplication(ctx, &apptypes.MsgStakeApplication{
		Address:  srcBech32,
		Stake:    &stake,
		Services: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
	})
	require.NoError(t, err)

	higherStake := apptypes.DefaultMinStake.AddAmount(math.NewInt(1000000))
	_, err = srv.StakeApplication(ctx, &apptypes.MsgStakeApplication{
		Address:  srcBech32,
		Stake:    &higherStake,
		Services: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc2"}},
	})
	require.NoError(t, err)

	srcBeforeTransfer, ok := k.GetApplication(ctx, srcBech32)
	require.True(t, ok)
	require.NotEmpty(t, srcBeforeTransfer.ServiceConfigHistory,
		"test setup: src must have non-empty history before transfer")
	for _, entry := range srcBeforeTransfer.ServiceConfigHistory {
		require.Equal(t, srcBech32, entry.ApplicationAddress,
			"test setup: src history entries must reference src address")
	}

	// Initiate transfer to a NEW address (dst does not yet exist).
	_, err = srv.TransferApplication(ctx, apptypes.NewMsgTransferApplication(srcBech32, dstBech32))
	require.NoError(t, err)

	sharedParams := sharedtypes.DefaultParams()
	srcApp, ok := k.GetApplication(ctx, srcBech32)
	require.True(t, ok)
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &srcApp)
	transferCompletionHeight := sharedtypes.GetSessionEndHeight(&sharedParams, transferEndHeight)
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(transferCompletionHeight)

	require.NoError(t, k.EndBlockerTransferApplication(ctx))

	// Every history entry on the destination must name the destination address.
	dstApp, isDstFound := k.GetApplication(ctx, dstBech32)
	require.True(t, isDstFound)
	require.NotEmpty(t, dstApp.ServiceConfigHistory,
		"dst must inherit non-empty history from src")
	for _, entry := range dstApp.ServiceConfigHistory {
		require.Equal(t, dstBech32, entry.ApplicationAddress,
			"dst history entry must name the dst address after transfer; got entry for service %q with ApplicationAddress=%q",
			entry.GetService().GetServiceId(), entry.ApplicationAddress)
	}

	// src is removed by the transfer EndBlocker.
	_, isSrcFound := k.GetApplication(ctx, srcBech32)
	require.False(t, isSrcFound, "src must be removed after a completed transfer")
}
