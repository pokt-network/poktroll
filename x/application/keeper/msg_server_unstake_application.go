package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// TODO_BETA(@bryanchriswhite): Determine if an application needs an unbonding period after unstaking.
func (k msgServer) UnstakeApplication(
	ctx context.Context,
	msg *apptypes.MsgUnstakeApplication,
) (*apptypes.MsgUnstakeApplicationResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"unstake_application",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UnstakeApplication")
	logger.Info(fmt.Sprintf("About to unstake application with msg: %v", msg))

	// Check if the application already exists or not.
	foundApp, isAppFound := k.GetApplication(ctx, msg.GetAddress())
	if !isAppFound {
		logger.Info(fmt.Sprintf("Application not found. Cannot unstake address (%s)", msg.GetAddress()))
		return nil, apptypes.ErrAppNotFound.Wrapf("application (%s)", msg.GetAddress())
	}
	logger.Info(fmt.Sprintf("Application found. Unstaking application for address (%s)", msg.GetAddress()))

	// Check if the application has already initiated the unstaking process.
	if foundApp.IsUnbonding() {
		logger.Warn(fmt.Sprintf("Application (%s) is still unbonding from previous unstaking", msg.GetAddress()))
		return nil, apptypes.ErrAppIsUnstaking.Wrapf("application (%s)", msg.GetAddress())
	}

	// Check if the application has already initiated a transfer process.
	// Transferring applications CANNOT unstake.
	if foundApp.HasPendingTransfer() {
		logger.Warn(fmt.Sprintf(
			"Application (%s) has a pending transfer to (%s)",
			msg.Address, foundApp.GetPendingTransfer().GetDestinationAddress()),
		)
		return nil, apptypes.ErrAppHasPendingTransfer.Wrapf("application (%s)", msg.GetAddress())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight)

	// Mark the application as unstaking by recording the height at which it should
	// no longer be able to request services.
	// The application MAY continue to request service until the end of the current
	// session. After that, the application will be considered inactive.
	foundApp.UnstakeSessionEndHeight = uint64(sessionEndHeight)
	k.SetApplication(ctx, foundApp)

	sdkCtx = sdk.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &foundApp)
	unbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
		Application:        &foundApp,
		Reason:             apptypes.ApplicationUnbondingReason_ELECTIVE,
		SessionEndHeight:   sessionEndHeight,
		UnbondingEndHeight: unbondingEndHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(unbondingBeginEvent); err != nil {
		err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", unbondingBeginEvent, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &apptypes.MsgUnstakeApplicationResponse{}, nil
}
