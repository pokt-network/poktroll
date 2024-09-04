package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
)

// TransferApplication transfers the stake (held in escrow in the application
// module account) from a source to a (new) destination application account .
func (k msgServer) TransferApplication(ctx context.Context, msg *types.MsgTransferApplication) (*types.MsgTransferApplicationResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"transfer_application_begin",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "TransferApplication")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Ensure destination application does not already exist.
	_, isDstFound := k.GetApplication(ctx, msg.GetDestinationAddress())
	if isDstFound {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrAppDuplicateAddress.Wrapf("destination application (%s) exists", msg.GetDestinationAddress()).Error(),
		)
	}

	// Ensure source application exists.
	srcApp, isAppFound := k.GetApplication(ctx, msg.GetSourceAddress())
	if !isAppFound {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrAppNotFound.Wrapf("source application (%s) not found", msg.GetSourceAddress()).Error(),
		)
	}

	// Ensure source application is not already unbonding.
	// TODO_TEST: Add E2E coverage to assert that an unbonding app cannot be transferred.
	if srcApp.IsUnbonding() {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrAppIsUnstaking.Wrapf("cannot transfer stake of unbonding source application %q", msg.GetSourceAddress()).Error(),
		)
	}

	// TODO_TEST: add E2E coverage to assert #DelegateeGatewayAddresses and #PendingUndelegations
	// are present and correct on the dstApp application.

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdkCtx.BlockHeight())

	srcApp.PendingTransfer = &types.PendingTransfer{
		Destination:      msg.GetDestinationAddress(),
		SessionEndHeight: uint64(sessionEndHeight),
	}

	// Update the dstApp in the store
	k.SetApplication(ctx, srcApp)
	logger.Info(fmt.Sprintf(
		"Successfully began transfer of application stake from (%s) to (%s)",
		srcApp.Address, msg.GetDestinationAddress(),
	))

	if err := sdkCtx.EventManager().EmitTypedEvent(&types.EventTransferBegin{
		SourceAddress:      srcApp.GetAddress(),
		DestinationAddress: srcApp.GetPendingTransfer().GetDestination(),
		SourceApplication:  &srcApp,
	}); err != nil {

	}

	isSuccessful = true

	return &types.MsgTransferApplicationResponse{
		Application: &srcApp,
	}, nil
}
