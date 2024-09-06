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

		// Transfer the stake of the source application to the destination application.
		if transferErr := k.transferApplication(ctx, srcApp); transferErr != nil {
			logger.Warn(transferErr.Error())

			// Remove the pending transfer from the source application.
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
// to a newly created destination application whose address is set to the destination address
// in the pending transfer of srcApp. It is intended to be called during the EndBlock ABCI method.
func (k Keeper) transferApplication(ctx context.Context, srcApp types.Application) error {
	logger := k.Logger().With("method", "transferApplication")

	// Double-check that the source application is not unbonding.
	// NB: This SHOULD NOT be possible because applications SHOULD NOT be able
	// to unstake when they have a pending transfer.
	if srcApp.IsUnbonding() {
		return types.ErrAppIsUnstaking.Wrapf("cannot transfer stake of unbonding source application (%s)", srcApp.GetAddress())
	}

	// Ensure destination application was not staked during transfer period.
	_, isDstFound := k.GetApplication(ctx, srcApp.GetPendingTransfer().GetDestinationAddress())
	if isDstFound {
		return types.ErrAppDuplicateAddress.Wrapf(
			"destination application (%s) was staked during transfer period of application (%s)",
			srcApp.GetPendingTransfer().GetDestinationAddress(), srcApp.GetAddress(),
		)
	}

	dstApp := srcApp // intentional copy
	dstApp.Address = srcApp.GetPendingTransfer().GetDestinationAddress()
	dstApp.PendingTransfer = nil

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdkCtx.BlockHeight())

	srcApp.PendingTransfer = &types.PendingApplicationTransfer{
		DestinationAddress: dstApp.Address,
		SessionEndHeight:   uint64(sessionEndHeight),
	}

	// Remove srcApp from the store
	k.RemoveApplication(ctx, srcApp.GetAddress())

	// Add the dstApp in the store
	k.SetApplication(ctx, dstApp)

	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.GetAddress(), dstApp.GetAddress()))

	if err := sdkCtx.EventManager().EmitTypedEvent(&types.EventTransferEnd{
		SourceAddress:          srcApp.GetAddress(),
		DestinationAddress:     dstApp.GetAddress(),
		DestinationApplication: &dstApp,
	}); err != nil {
		logger.Error(fmt.Sprintf("could not emit transfer end event: %v", err))
	}

	return nil
}
