package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO(#489): Determine if an application needs an unbonding period after unstaking.
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
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)

	// Mark the application as unstaking by recording the height at which it should
	// no longer be able to request services.
	// The application MAY continue to request service until the end of the current
	// session. After that, the application will be considered inactive.
	foundApp.UnstakeSessionEndHeight = uint64(sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight))
	k.SetApplication(ctx, foundApp)

	// TODO_UPNEXT:(@bryanchriswhite): emit a new EventApplicationUnbondingBegin event.

	isSuccessful = true
	return &apptypes.MsgUnstakeApplicationResponse{}, nil
}
