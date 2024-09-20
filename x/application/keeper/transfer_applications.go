package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
)

// EndBlockerTransferApplication completes pending application transfers at the
// end of the session in which they began by copying the current state of the source
// application onto a new destination application, unstaking (removing) the source,
// and staking (storing) the destination.
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
	logger := k.Logger().With("method", "EndBlockerTransferApplication")

	// Only process application transfers at the end of the session.
	if !shared.IsSessionEndHeight(&sharedParams, currentHeight) {
		return nil
	}

	// Iterate over all applications and transfer the ones that have finished the transfer period.
	// TODO_IMPROVE: Use an index to iterate over the applications that have initiated
	// the transfer action instead of iterating over all of them.
	for _, srcApp := range k.GetAllApplications(ctx) {
		// Ignore applications that have not initiated the transfer action.
		if !srcApp.HasPendingTransfer() {
			continue
		}

		// Ignore applications that have initiated a transfer but still active
		transferEndHeight := types.GetApplicationTransferHeight(&sharedParams, &srcApp)
		if sdkCtx.BlockHeight() < transferEndHeight {
			continue
		}

		// Transfer the stake of the source application to the destination application and
		// merge their gateway delegations and service configs.
		if transferErr := k.transferApplication(ctx, srcApp); transferErr != nil {
			logger.Warn(transferErr.Error())

			// Application transfer failed, removing the pending transfer from the source application.
			srcApp.PendingTransfer = nil
			k.SetApplication(ctx, srcApp)

			if err := sdkCtx.EventManager().EmitTypedEvent(&types.EventTransferError{
				SourceAddress:      srcApp.GetAddress(),
				DestinationAddress: srcApp.GetPendingTransfer().GetDestinationAddress(),
				SourceApplication:  &srcApp,
				Error:              transferErr.Error(),
			}); err != nil {
				logger.Error(fmt.Sprintf("could not emit transfer error event: %v", err))
			}
		}
	}

	isSuccessful = true
	return nil
}

// transferApplication transfers the fields of srcApp, except for address and pending_transfer,
// to an application whose address is the destination address of the pending transfer of srcApp.
// If the destination application does not exist, it is created. If it does exist, the stake of
// the destination application is incremented by the stake of the source application, and the
// delegatees and service configs of the destination application are set to the union of the
// source application's delegatees and service configs and the destination application's
// delegatees. It is intended to be called during the EndBlock ABCI method.
func (k Keeper) transferApplication(ctx context.Context, srcApp types.Application) error {
	logger := k.Logger().With("method", "transferApplication")

	// Double-check that the source application is not unbonding.
	// NB: This SHOULD NOT be possible because applications SHOULD NOT be able
	// to unstake when they have a pending transfer.
	if srcApp.IsUnbonding() {
		return types.ErrAppIsUnstaking.Wrapf("cannot transfer stake of unbonding source application (%s)", srcApp.GetAddress())
	}

	// Check if the destination application already exists. If not, derive it from
	// the source application. If so, "merge" the source application into the
	// destination by summing stake amounts and taking the union of delegations
	// and service configs.
	dstApp, isDstFound := k.GetApplication(ctx, srcApp.GetPendingTransfer().GetDestinationAddress())
	if !isDstFound {
		dstApp = srcApp //intentional copy
		dstApp.Address = srcApp.GetPendingTransfer().GetDestinationAddress()
		dstApp.PendingTransfer = nil
	} else {
		srcStakeSumCoin := dstApp.GetStake().Add(*dstApp.GetStake())
		dstApp.Stake = &srcStakeSumCoin

		mergeAppDelegatees(&srcApp, &dstApp)
		mergeAppServiceConfigs(&srcApp, &dstApp)
	}

	// Remove srcApp from the store
	k.RemoveApplication(ctx, srcApp.GetAddress())

	// Add or update the dstApp in the store
	k.SetApplication(ctx, dstApp)

	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.GetAddress(), dstApp.GetAddress()))

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	if err := sdkCtx.EventManager().EmitTypedEvent(&types.EventTransferEnd{
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
func mergeAppDelegatees(srcApp, dstApp *types.Application) {
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

// mergeAppServiceConfigs takes the union of the srcApp and dstApp's service configs
// and sets the result in dstApp.
func mergeAppServiceConfigs(srcApp, dstApp *types.Application) {
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
