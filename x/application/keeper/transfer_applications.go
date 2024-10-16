package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// EndBlockerTransferApplication completes pending application transfers.
// This always happens on the last block of a session during which the transfer started.
// It is accomplished by:
//  1. Copying the current state of the source app onto a new destination app
//  2. Unstaking (removing) the source app
//  3. Staking (storing) the destination app
func (k Keeper) EndBlockerTransferApplication(ctx context.Context) error {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"transfer_application_end",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	logger := k.Logger().
		With("method", "EndBlockerTransferApplication").
		With("current_height", currentHeight).
		With("session_end_height", sessionEndHeight)

	// Only process application transfers at the end of the session in
	// order to avoid inconsistent/unpredictable mid-session behavior.
	if !sharedtypes.IsSessionEndHeight(&sharedParams, currentHeight) {
		return nil
	}

	// Iterate over all applications and transfer the ones that have finished the transfer period.
	// TODO_MAINNET(@bryanchriswhite, #854): Use an index to iterate over the applications that have initiated
	// the transfer action instead of iterating over all of them.
	for _, srcApp := range k.GetAllApplications(ctx) {
		// Ignore applications that have not initiated the transfer action.
		if !srcApp.HasPendingTransfer() {
			continue
		}

		// Ignore applications that have initiated a transfer but still active.
		// This spans the period from the end of the session in which the transfer
		// began to the end of settlement for that session.
		transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &srcApp)
		if sdkCtx.BlockHeight() < transferEndHeight {
			continue
		}

		// Transfer the stake of the source application to the destination application and
		// merge their gateway delegations and service configs.
		if transferErr := k.transferApplication(ctx, srcApp); transferErr != nil {
			logger.Warn(transferErr.Error())

			// Application transfer failed, removing the pending transfer from the source application.
			dstBech32 := srcApp.GetPendingTransfer().GetDestinationAddress()
			srcApp.PendingTransfer = nil
			k.SetApplication(ctx, srcApp)

			if err := sdkCtx.EventManager().EmitTypedEvent(&apptypes.EventTransferError{
				SourceAddress:      srcApp.GetAddress(),
				DestinationAddress: dstBech32,
				SourceApplication:  &srcApp,
				Error:              transferErr.Error(),
			}); err != nil {
				return err
			}
		}
	}

	isSuccessful = true
	return nil
}

// transferApplication transfers the fields of srcApp, except for address and pending_transfer,
// to an application whose address is the destination address of the pending transfer of srcApp.
// If the destination application does not exist, it is created. If it does exist, the stake of
// the destination application stake is incremented by the stake of the source application, and the
// delegatees and service configs of the destination application are set to the union of the
// source and destination applications' delegatees and service configs. It is intended
// to be called during the EndBlock ABCI method.

// transferApplication transfers srcApp to srcApp.PendingTransfer.destination.
// If the destination application does not exist, it is created.
// If it does exist, then destination app is updated as follows:
//   - The application stake is incremented by the stake of the source application
//   - The delegatees and service configs of the destination application are set to the union of the src and dest
//   - The pending undelegations of the source are merged into the destination.
//     Duplicate pending undelegations resolve to the destination application's.
//
// It is intended to be called during the EndBlock ABCI method.
func (k Keeper) transferApplication(ctx context.Context, srcApp apptypes.Application) error {
	logger := k.Logger().With("method", "transferApplication")

	// Double-check that the source application is not unbonding.
	// NB: This SHOULD NOT be possible because applications SHOULD NOT be able
	// to unstake when they have a pending transfer.
	if srcApp.IsUnbonding() {
		return apptypes.ErrAppIsUnstaking.Wrapf("cannot transfer stake of unbonding source application (%s)", srcApp.GetAddress())
	}

	// Check if the destination application already exists:
	// If it does not: derive it from the source application.
	// If it does: "merge" the src app into the dst app by:
	// - summing stake amounts
	// - taking the union of delegations and service configs
	dstApp, isDstFound := k.GetApplication(ctx, srcApp.GetPendingTransfer().GetDestinationAddress())
	if !isDstFound {
		dstApp = srcApp //intentional copy
		dstApp.Address = srcApp.GetPendingTransfer().GetDestinationAddress()
		dstApp.PendingTransfer = nil

		logger.Info(fmt.Sprintf(
			"transferring application from %q to new application %q",
			srcApp.GetAddress(), dstApp.GetAddress(),
		))
	} else {
		srcStakeSumCoin := dstApp.GetStake().Add(*dstApp.GetStake())
		dstApp.Stake = &srcStakeSumCoin

		mergeAppDelegatees(&srcApp, &dstApp)
		mergeAppPendingUndelegations(&srcApp, &dstApp)
		mergeAppServiceConfigs(&srcApp, &dstApp)

		logger.Info(fmt.Sprintf(
			"transferring application from %q to existing application %q",
			srcApp.GetAddress(), dstApp.GetAddress(),
		))
	}

	// Remove srcApp from the store
	k.RemoveApplication(ctx, srcApp.GetAddress())

	// Add or update the dstApp in the store
	k.SetApplication(ctx, dstApp)

	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.GetAddress(), dstApp.GetAddress()))

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	if err := sdkCtx.EventManager().EmitTypedEvent(&apptypes.EventTransferEnd{
		SourceAddress:          srcApp.GetAddress(),
		DestinationAddress:     dstApp.GetAddress(),
		DestinationApplication: &dstApp,
	}); err != nil {
		logger.Error(fmt.Sprintf("could not emit transfer end event: %v", err))
	}

	return nil
}

// mergeAppDelegatees takes the union of the srcApp and dstApp's delegatees and
// sets the result in dstApp.
func mergeAppDelegatees(srcApp, dstApp *apptypes.Application) {
	// Build a set of the destination application's delegatees.
	delagateeBech32Set := make(map[string]struct{})
	for _, dstDelegateeBech32 := range dstApp.DelegateeGatewayAddresses {
		delagateeBech32Set[dstDelegateeBech32] = struct{}{}
	}

	// Build the union of the source and destination applications' delagatees by
	// appending source application delegatees which are not already in the set.
	for _, srcDelegateeBech32 := range srcApp.DelegateeGatewayAddresses {
		if _, ok := delagateeBech32Set[srcDelegateeBech32]; !ok {
			dstApp.DelegateeGatewayAddresses = append(dstApp.DelegateeGatewayAddresses, srcDelegateeBech32)
		}
	}
}

// mergeAppPendingUndelegations takes the union of the srcApp and dstApp's pending undelegations
// and sets the result in dstApp. Pending undelegations are merged according to the following algorithm:
// - At each pending undelegation height in the destination application:
//   - Take the union of the gateway addresses in source and destination applications'
//     pending undelegations at that height.
//   - If a gateway address is present in both source and destination application's pending undelegations
//     **at different heights**, the destination application's undelegation height is used
//     (i.e. this undelegation is unchanged) and that gateway address is excluded from the
//     gateway address union at the height which it is present in the source application's
//     pending undelegations.
func mergeAppPendingUndelegations(srcApp, dstApp *apptypes.Application) {
	// Build a map from all gateway addresses which have pending undelegations to
	// their respective undelegation session end heights. If the source and destination
	// applications both contain pending undelegations from the same gateway address, the
	// source pending undelegation is ignored. E.g., srcApp has a pending undelegation at
	// height 10 from gateway1 and dstApp has pending undelegation from gateway1 atheight
	// 20; only one pending undelegation from gateway1 should be present in the merged
	// result, at height 20.
	undelegationsUnionAddrToHeightMap := make(map[string]uint64)

	for srcHeight, srcUndelegetingGatewaysList := range srcApp.PendingUndelegations {
		for _, srcGateway := range srcUndelegetingGatewaysList.GatewayAddresses {
			undelegationsUnionAddrToHeightMap[srcGateway] = srcHeight
		}
	}

	for dstHeight, dstUndelegatingGatewaysList := range dstApp.PendingUndelegations {
		for _, dstGateway := range dstUndelegatingGatewaysList.GatewayAddresses {
			undelegationsUnionAddrToHeightMap[dstGateway] = dstHeight
		}
	}

	// Reset the destination application's pending undelegations and rebuild it
	// from the undelegations union map.
	dstApp.PendingUndelegations = make(map[uint64]apptypes.UndelegatingGatewayList)
	for gatewayAddr, height := range undelegationsUnionAddrToHeightMap {
		dstPendingUndelegationsAtHeight, ok := dstApp.PendingUndelegations[height]
		if !ok {
			dstApp.PendingUndelegations[height] = apptypes.UndelegatingGatewayList{
				GatewayAddresses: []string{gatewayAddr},
			}
		} else {
			dstPendingUndelegationsAtHeight.GatewayAddresses = append(dstApp.PendingUndelegations[height].GatewayAddresses, gatewayAddr)
			dstApp.PendingUndelegations[height] = dstPendingUndelegationsAtHeight
		}
	}
}

// mergeAppServiceConfigs takes the union of the srcApp and dstApp's service configs
// and sets the result in dstApp.
func mergeAppServiceConfigs(srcApp, dstApp *apptypes.Application) {
	// Build a set of the destination application's service configs.
	serviceIDSet := make(map[string]struct{})
	for _, dstServiceConfig := range dstApp.ServiceConfigs {
		serviceIDSet[dstServiceConfig.GetServiceId()] = struct{}{}
	}

	// Build the union of the source and destination applications' service configs by
	// appending source application service configs which are not already in the set.
	for _, srcServiceConfig := range srcApp.ServiceConfigs {
		if _, ok := serviceIDSet[srcServiceConfig.GetServiceId()]; !ok {
			dstApp.ServiceConfigs = append(dstApp.ServiceConfigs, srcServiceConfig)
		}
	}
}
