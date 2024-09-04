package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
)

func (k Keeper) EndBlockerTransferApplication(ctx context.Context) error {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"transfer_application_stake_complete",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	logger := k.Logger().With("method", "EndBlockerTransferApplication")

	// Only process application transfers at the end of the session.
	// TODO_CONSIDER: refactoring this logic into a shared function, `IsCurrentSessionEndHeight`.
	currentSessionEndHeight := shared.GetSessionEndHeight(&sharedParams, currentHeight)
	if currentHeight != currentSessionEndHeight {
		return nil
	}

	// Iterate over all applications and transfer the ones that have finished the transfer period.
	// TODO_IMPROVE: Use an index to iterate over the applications that have initiated
	// the transfer action instead of iterating over all of them.
	var transferredSrcApps []types.Application
	for _, srcApp := range k.GetAllApplications(ctx) {
		// Ignore applications that have not initiated the transfer action.
		if !srcApp.HasPendingTransfer() {
			continue
		}

		transferCompleteHeight := types.GetApplicationTransferHeight(&sharedParams, &srcApp)
		if sdkCtx.BlockHeight() < transferCompleteHeight {
			continue
		}

		// Transfer the stake of the source application to the destination application.
		if transferErr := k.transferApplication(ctx, srcApp); transferErr != nil {
			logger.Warn(transferErr.Error())
			continue
		}

		transferredSrcApps = append(transferredSrcApps, srcApp)
	}

	// Remove the transferred apps from the store
	for _, srcApp := range transferredSrcApps {
		k.RemoveApplication(ctx, srcApp.GetAddress())
		logger.Info(fmt.Sprintf(
			"Successfully completed transfer of application stake from (%s) to (%s)",
			srcApp.GetAddress(), srcApp.GetPendingTransfer().GetDestination(),
		))
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
	_, isDstFound := k.GetApplication(ctx, srcApp.GetPendingTransfer().GetDestination())
	if isDstFound {
		return types.ErrAppDuplicateAddress.Wrapf(
			"destination application (%s) was staked during transfer period of application (%s)",
			srcApp.GetPendingTransfer().GetDestination(), srcApp.GetAddress(),
		)
	}

	dstApp := srcApp
	dstApp.Address = srcApp.GetPendingTransfer().GetDestination()
	dstApp.PendingTransfer = nil

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdkCtx.BlockHeight())

	srcApp.PendingTransfer = &types.PendingTransfer{
		Destination:      dstApp.Address,
		SessionEndHeight: uint64(sessionEndHeight),
	}

	// Remove srcApp from the store
	k.RemoveApplication(ctx, srcApp.GetAddress())

	// Add the dstApp in the store
	k.SetApplication(ctx, dstApp)

	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.GetAddress(), dstApp.GetAddress()))

	// TODO_IN_THIS_PR: add and emit an ApplicationTransferEndEvent.

	return nil
}
