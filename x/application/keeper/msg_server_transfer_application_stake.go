package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
)

// TransferApplication transfers the stake (held in escrow in the application
// module account) from a source to a (new) destination application account .
func (k msgServer) TransferApplication(ctx context.Context, msg *types.MsgTransferApplication) (*types.MsgTransferApplicationResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"transfer_application_stake",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "TransferApplication")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Ensure destination application does not already exist.
	_, isDstFound := k.GetApplication(ctx, msg.GetDestinationAddress())
	if isDstFound {
		return nil, types.ErrAppDuplicateAddress.Wrapf("destination application (%q) exists", msg.GetDestinationAddress())
	}

	// Ensure source application exists.
	srcApp, isAppFound := k.GetApplication(ctx, msg.GetSourceAddress())
	if !isAppFound {
		return nil, types.ErrAppNotFound.Wrapf("source application %q not found", msg.GetSourceAddress())
	}

	// Ensure source application is not already unbonding.
	// TODO_TEST: Add E2E coverage to assert that an unbonding app cannot be transferred.
	if srcApp.IsUnbonding() {
		return nil, types.ErrAppIsUnstaking.Wrapf("cannot transfer stake of unbonding source application %q", msg.GetSourceAddress())
	}

	// Create a new application derived from the source application.
	dstApp := srcApp
	dstApp.Address = msg.GetDestinationAddress()

	// TODO_IN_THIS_PR: Reconcile app unbonding logic with the new transfer stake logic.
	// I.e., the source should not immediately be transferred.

	// TODO_TEST: add E2E coverage to assert #DelegateeGatewayAddresses and #PendingUndelegations
	// are present and correct on the dstApp application.

	// Update the dstApp in the store
	k.SetApplication(ctx, dstApp)
	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.Address, dstApp.Address))

	// Remove the transferred app from the store
	k.RemoveApplication(ctx, srcApp.GetAddress())
	logger.Info(fmt.Sprintf("Successfully removed the application: %+v", srcApp))

	isSuccessful = true

	return &types.MsgTransferApplicationResponse{
		Application: &srcApp,
	}, nil
}
