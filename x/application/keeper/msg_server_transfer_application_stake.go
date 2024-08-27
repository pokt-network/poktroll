package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
)

// TransferApplicationStake transfers the stake (held in escrow in the application
// module account) from a source to a (new) destination application account .
func (k msgServer) TransferApplicationStake(ctx context.Context, msg *types.MsgTransferApplicationStake) (*types.MsgTransferApplicationStakeResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"transfer_application_stake",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "TransferApplicationStake")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, isDstFound := k.GetApplication(ctx, msg.GetDestinationAddress())
	if isDstFound {
		return nil, types.ErrAppDuplicateAddress.Wrapf("destination application (%q) exists", msg.GetDestinationAddress())
	}

	srcApp, isAppFound := k.GetApplication(ctx, msg.GetSourceAddress())
	if !isAppFound {
		return nil, types.ErrAppNotFound.Wrapf("source application %q not found", msg.GetSourceAddress())
	}

	// Create a new application derived from the source application.
	dstApp := srcApp
	dstApp.Address = msg.GetDestinationAddress()

	// TODO_TEST: add E2E coverage to assert #DelegateeGatewayAddresses and #PendingUndelegations
	// are present on the dstApp application.

	// Update the dstApp in the store
	k.SetApplication(ctx, dstApp)
	logger.Info(fmt.Sprintf("Successfully transferred application stake from (%s) to (%s)", srcApp.Address, dstApp.Address))

	// Remove the transferred app from the store
	k.RemoveApplication(ctx, srcApp.GetAddress())
	logger.Info(fmt.Sprintf("Successfully removed the application: %+v", srcApp))

	isSuccessful = true

	return &types.MsgTransferApplicationStakeResponse{
		Application: &dstApp,
	}, nil
}
