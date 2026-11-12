package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// MigrateNonCanonicalApplications handles application addresses stored in a
// non-canonical bech32 encoding (accepted before application messages required
// the canonical form):
//   - an application stored under a non-canonical address begins unbonding, as
//     MsgUnstakeApplication would, and its pending transfer (if any) is dropped.
//     No message can address it anymore, so without this its stake could never
//     be released. The refund goes to the same account bytes;
//   - a non-canonical pending transfer destination is rewritten to its canonical
//     form, so the transfer creates the canonical application. A transfer whose
//     canonical destination is the source itself, or an application that already
//     exists, is dropped instead. Every dropped transfer is reported as a failed
//     transfer (EventTransferError).
//
// Addresses that do not decode are skipped: unbonding refunds to the decoded
// address, so marking them unbonding would fail the unbonding EndBlocker.
// Every key written is the stored string, so an application stored under the
// canonical encoding of the same account is never touched by the first rule.
func (k Keeper) MigrateNonCanonicalApplications(ctx context.Context) (numUnbondingBegun, numTransferDestinationsCanonicalized, numTransfersDropped int) {
	logger := k.Logger().With("method", "MigrateNonCanonicalApplications")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdkCtx.BlockHeight())

	// GetAllApplications collects every application before the loop below writes.
	for _, app := range k.GetAllApplications(ctx) {
		if isNonCanonicalAddress(app.Address) {
			if app.IsUnbonding() {
				continue
			}
			if app.HasPendingTransfer() {
				k.dropPendingTransfer(sdkCtx, &app, sessionEndHeight, "source application address is not canonical")
				numTransfersDropped++
			}
			app.UnstakeSessionEndHeight = uint64(sessionEndHeight)
			k.SetApplication(ctx, app)
			numUnbondingBegun++
			logger.Info(fmt.Sprintf("began unbonding application with non-canonical address %s", app.Address))

			unbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
				Application:        &app,
				Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
				SessionEndHeight:   sessionEndHeight,
				UnbondingEndHeight: k.GetApplicationUnbondingEndHeight(ctx, &app),
			}
			if err := sdkCtx.EventManager().EmitTypedEvent(unbondingBeginEvent); err != nil {
				logger.Error(fmt.Sprintf("failed to emit %T for application %s: %s", unbondingBeginEvent, app.Address, err))
			}
			continue
		}

		if !app.HasPendingTransfer() {
			continue
		}
		dstAddress, err := cosmostypes.AccAddressFromBech32(app.PendingTransfer.DestinationAddress)
		if err != nil || dstAddress.String() == app.PendingTransfer.DestinationAddress {
			continue
		}
		_, dstExists := k.GetApplication(ctx, dstAddress.String())
		switch {
		case dstAddress.String() == app.Address:
			k.dropPendingTransfer(sdkCtx, &app, sessionEndHeight, "destination is the source application")
			numTransfersDropped++
		case dstExists:
			k.dropPendingTransfer(sdkCtx, &app, sessionEndHeight, "destination application already exists")
			numTransfersDropped++
		default:
			logger.Info(fmt.Sprintf("canonicalized pending transfer destination %s of application %s", app.PendingTransfer.DestinationAddress, app.Address))
			app.PendingTransfer.DestinationAddress = dstAddress.String()
			numTransferDestinationsCanonicalized++
		}
		k.SetApplication(ctx, app)
	}

	return numUnbondingBegun, numTransferDestinationsCanonicalized, numTransfersDropped
}

// dropPendingTransfer clears an application's pending transfer and reports it as
// a failed transfer, so a transfer that began with EventTransferBegin also ends.
func (k Keeper) dropPendingTransfer(sdkCtx cosmostypes.Context, app *apptypes.Application, sessionEndHeight int64, reason string) {
	logger := k.Logger().With("method", "MigrateNonCanonicalApplications")
	dstAddress := app.PendingTransfer.DestinationAddress
	app.PendingTransfer = nil
	logger.Info(fmt.Sprintf("dropped pending transfer of application %s to %s: %s", app.Address, dstAddress, reason))

	transferErrorEvent := &apptypes.EventTransferError{
		SourceAddress:      app.Address,
		DestinationAddress: dstAddress,
		SourceApplication:  app,
		SessionEndHeight:   sessionEndHeight,
		Error:              reason,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(transferErrorEvent); err != nil {
		logger.Error(fmt.Sprintf("failed to emit %T for application %s: %s", transferErrorEvent, app.Address, err))
	}
}

// isNonCanonicalAddress reports whether address decodes but is not in its
// canonical bech32 form (e.g. its all-uppercase alias).
func isNonCanonicalAddress(address string) bool {
	accAddress, err := cosmostypes.AccAddressFromBech32(address)
	return err == nil && accAddress.String() != address
}
