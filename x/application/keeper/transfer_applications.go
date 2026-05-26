package keeper

import (
	"context"
	"fmt"
	"sort"

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

	// Iterate over all applications that have a pending transfer.
	// Transfer the ones that have finished the transfer period.
	allTransferringApplicationsIterator := k.GetAllTransferringApplicationsIterator(ctx)
	defer allTransferringApplicationsIterator.Close()

	for ; allTransferringApplicationsIterator.Valid(); allTransferringApplicationsIterator.Next() {
		srcApp, err := allTransferringApplicationsIterator.Value()
		if err != nil {
			return err
		}

		// Ignore applications that have not initiated the transfer action.
		if !srcApp.HasPendingTransfer() {
			// If we are getting the application from the transfer store and it is not
			// transferring, this means that there is a dangling entry in the index.
			// log the error, remove the index entry but continue to the next application.
			logger.Error(fmt.Sprintf(
				"found application %s in transfer store but it is not transferring, removing index entry",
				srcApp.Address,
			))
			k.removeApplicationTransferIndex(ctx, srcApp.Address)
			continue
		}

		// Ignore applications that have initiated a transfer but still active.
		// This spans the period from the end of the session in which the transfer
		// began to the end of settlement for that session.
		transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &srcApp)
		if currentHeight < transferEndHeight {
			continue
		}

		// Transfer the stake of the source application to the destination application and
		// merge their gateway delegations and service configs.
		if err := k.transferApplication(ctx, srcApp); err != nil {
			logger.Warn(err.Error())

			// Application transfer failed, removing the pending transfer from the source application.
			dstBech32 := srcApp.GetPendingTransfer().GetDestinationAddress()
			srcApp.PendingTransfer = nil
			k.SetApplication(ctx, srcApp)

			transferErrorEvent := &apptypes.EventTransferError{
				SourceAddress:      srcApp.GetAddress(),
				DestinationAddress: dstBech32,
				SourceApplication:  &srcApp,
				SessionEndHeight:   sessionEndHeight,
				Error:              err.Error(),
			}
			if err = sdkCtx.EventManager().EmitTypedEvent(transferErrorEvent); err != nil {
				err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", transferErrorEvent, err)
				logger.Error("%s", err)
				return err
			}
		}
	}

	isSuccessful = true
	return nil
}

// transferApplication transfers srcApp to srcApp.PendingTransfer.destination.
// If the destination application does not exist, it is created.
// If it does exist, then destination app is updated as follows:
//   - The application stake is incremented by the stake of the source application
//   - The delegatees and service configs of the destination application are set to the union of the src and dest
//   - The pending undelegations of the source are merged into the destination.
//     Duplicate pending undelegations resolve to the destination application's.
//
// It is intended to be called during the EndBlock ABCI method.
func (k Keeper) transferApplication(
	ctx context.Context,
	srcApp apptypes.Application,
) error {
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

	// finalServiceConfigs is the destination's service config set after the
	// transfer. It is computed WITHOUT pre-mutating dstApp.ServiceConfigs so that
	// recordApplicationServiceConfigChange below can compare the destination's
	// PRIOR set (still on dstApp) against this new set and correctly record a
	// service swap when the merge adds services.
	var finalServiceConfigs []*sharedtypes.ApplicationServiceConfig
	if !isDstFound {
		dstApp = srcApp //intentional copy
		dstApp.Address = srcApp.GetPendingTransfer().GetDestinationAddress()
		dstApp.PendingTransfer = nil
		finalServiceConfigs = dstApp.ServiceConfigs

		// Rewrite ApplicationAddress on each copied history entry to the new
		// dst address. The shallow copy above aliased dstApp.ServiceConfigHistory
		// with srcApp.ServiceConfigHistory (both slices point at the same
		// *ApplicationServiceConfigUpdate values), and each entry's
		// ApplicationAddress still names the SOURCE app. Without this rewrite,
		// indexers and queries that filter history by app address would miss the
		// dst app's pre-transfer entries. Allocate a fresh slice with cloned
		// entries to avoid mutating srcApp's view (srcApp is removed below, but
		// clarity is worth the small alloc).
		if len(dstApp.ServiceConfigHistory) > 0 {
			rewrittenHistory := make([]*apptypes.ApplicationServiceConfigUpdate, 0, len(dstApp.ServiceConfigHistory))
			for _, entry := range dstApp.ServiceConfigHistory {
				if entry == nil {
					continue
				}
				cloned := *entry
				cloned.ApplicationAddress = dstApp.Address
				rewrittenHistory = append(rewrittenHistory, &cloned)
			}
			dstApp.ServiceConfigHistory = rewrittenHistory
		}

		logger.Info(fmt.Sprintf(
			"transferring application from %q to new application %q",
			srcApp.GetAddress(), dstApp.GetAddress(),
		))
	} else {
		srcStakeSumCoin := srcApp.GetStake().Add(*dstApp.GetStake())
		dstApp.Stake = &srcStakeSumCoin

		mergeAppDelegatees(&srcApp, &dstApp)
		mergeAppPendingUndelegations(&srcApp, &dstApp)
		// NB: do NOT mutate dstApp.ServiceConfigs here — the merged union is passed
		// to recordApplicationServiceConfigChange, which needs dstApp to still hold
		// the prior set to detect the change and append history entries.
		finalServiceConfigs = mergeAppServiceConfigs(&srcApp, &dstApp)
		mergeAppPerSessionSpendLimit(&srcApp, &dstApp)

		logger.Info(fmt.Sprintf(
			"transferring application from %q to existing application %q",
			srcApp.GetAddress(), dstApp.GetAddress(),
		))
	}

	// Synchronize the destination's service_config_history with its final
	// (possibly merged) ServiceConfigs. recordApplicationServiceConfigChange is a
	// no-op when the service set is unchanged (e.g. transfer to a new address with
	// the same single service — history stays empty and GetActiveServiceConfigs
	// falls back to the flat snapshot), and records a session-boundary change when
	// the merge altered the service set, so session hydration (which reads history
	// once non-empty) can resolve all of the destination's services. It also keeps
	// dstApp.ServiceConfigs in sync with finalServiceConfigs.
	k.recordApplicationServiceConfigChange(ctx, &dstApp, finalServiceConfigs)

	// Remove srcApp from the store
	k.RemoveApplication(ctx, srcApp)

	// Add or update the dstApp in the store
	k.SetApplication(ctx, dstApp)

	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.GetAddress(), dstApp.GetAddress()))

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &srcApp)
	transferEndEvent := &apptypes.EventTransferEnd{
		SourceAddress:          srcApp.GetAddress(),
		DestinationAddress:     dstApp.GetAddress(),
		DestinationApplication: &dstApp,
		SessionEndHeight:       sessionEndHeight,
		TransferEndHeight:      transferEndHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(transferEndEvent); err != nil {
		err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", transferEndEvent, err)
		logger.Error(err.Error())
		return err
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

	// CONSENSUS-CRITICAL: Sort gateway addresses within each height for deterministic
	// protobuf serialization. The loop above iterates undelegationsUnionAddrToHeightMap
	// (a Go map with non-deterministic iteration order), so the GatewayAddresses slices
	// are in arbitrary order. Without this sort, different validators produce different
	// serialized state, causing AppHash mismatch and chain halt.
	for height, list := range dstApp.PendingUndelegations {
		sort.Strings(list.GatewayAddresses)
		dstApp.PendingUndelegations[height] = list
	}
}

// mergeAppServiceConfigs takes the union of the srcApp and dstApp's service configs
// and sets the result in dstApp.
// mergeAppServiceConfigs returns the union of the source and destination
// applications' service configs, preserving the destination's existing order
// followed by any source-only configs. It does NOT mutate dstApp.ServiceConfigs;
// the caller passes the returned union to recordApplicationServiceConfigChange,
// which needs dstApp to still hold its prior set to detect the change.
func mergeAppServiceConfigs(srcApp, dstApp *apptypes.Application) []*sharedtypes.ApplicationServiceConfig {
	// Build a set of the destination application's service configs.
	serviceIDSet := make(map[string]struct{})
	for _, dstServiceConfig := range dstApp.ServiceConfigs {
		serviceIDSet[dstServiceConfig.GetServiceId()] = struct{}{}
	}

	// Start from the destination's existing configs, then append source-only ones.
	merged := make([]*sharedtypes.ApplicationServiceConfig, 0, len(dstApp.ServiceConfigs)+len(srcApp.ServiceConfigs))
	merged = append(merged, dstApp.ServiceConfigs...)
	for _, srcServiceConfig := range srcApp.ServiceConfigs {
		if _, ok := serviceIDSet[srcServiceConfig.GetServiceId()]; !ok {
			merged = append(merged, srcServiceConfig)
		}
	}

	return merged
}

// mergeAppPerSessionSpendLimit merges the per-session spend limits of the source
// and destination applications during a transfer. If both apps have a limit, the
// more restrictive (lower) limit is used. If only one has a limit, that limit is
// preserved. If neither has a limit, the result has no limit.
func mergeAppPerSessionSpendLimit(srcApp, dstApp *apptypes.Application) {
	srcLimit := srcApp.PerSessionSpendLimit
	dstLimit := dstApp.PerSessionSpendLimit

	switch {
	case srcLimit == nil && dstLimit == nil:
		// Neither has a limit — nothing to do.
	case srcLimit != nil && dstLimit == nil:
		// Only source has a limit — adopt it.
		dstApp.PerSessionSpendLimit = srcLimit
	case srcLimit == nil && dstLimit != nil:
		// Only destination has a limit — keep it (already set).
	default:
		// Both have limits — take the more restrictive (lower) one.
		if srcLimit.Amount.LT(dstLimit.Amount) {
			dstApp.PerSessionSpendLimit = srcLimit
		}
	}
}
