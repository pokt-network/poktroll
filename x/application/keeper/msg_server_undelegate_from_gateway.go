package keeper

import (
	"context"
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UndelegateFromGateway(ctx context.Context, msg *apptypes.MsgUndelegateFromGateway) (*apptypes.MsgUndelegateFromGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"undelegate_from_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UndelegateFromGateway")
	logger.Info(fmt.Sprintf("About to undelegate application from gateway with msg: %+v", msg))

	// Basic validation of the message
	if err := msg.ValidateBasic(); err != nil {
		logger.Info(fmt.Sprintf("Undelegation Message failed basic validation: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Retrieve the application from the store
	foundApp, isAppFound := k.GetApplication(ctx, msg.GetAppAddress())
	if !isAppFound {
		logger.Info(fmt.Sprintf("Application not found with address [%s]", msg.GetAppAddress()))
		return nil, status.Error(
			codes.NotFound,
			apptypes.ErrAppNotFound.Wrapf(
				"application with address: %q", msg.GetAppAddress(),
			).Error(),
		)
	}
	logger.Info(fmt.Sprintf("Application found with address [%s]", msg.GetAppAddress()))

	// Check if the application is already delegated to the gateway
	foundIdx := -1
	for i, gatewayAddr := range foundApp.DelegateeGatewayAddresses {
		if gatewayAddr == msg.GetGatewayAddress() {
			foundIdx = i
		}
	}
	if foundIdx == -1 {
		logger.Info(fmt.Sprintf("Application not delegated to gateway with address [%s]", msg.GetGatewayAddress()))
		return nil, status.Error(
			codes.FailedPrecondition,
			apptypes.ErrAppNotDelegated.Wrapf(
				"application with address %q not delegated to gateway with address: %q",
				msg.GetAppAddress(), msg.GetGatewayAddress(),
			).Error(),
		)
	}

	// Remove the gateway from the application's delegatee gateway public keys
	foundApp.DelegateeGatewayAddresses = append(foundApp.DelegateeGatewayAddresses[:foundIdx], foundApp.DelegateeGatewayAddresses[foundIdx+1:]...)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight)

	k.recordPendingUndelegation(ctx, &foundApp, msg.GetGatewayAddress(), currentHeight)

	// Update the application store with the new delegation
	k.SetApplication(ctx, foundApp)
	logger.Info(fmt.Sprintf("Successfully undelegated application from gateway for app: %+v", foundApp))

	// Emit the application redelegation event
	event := &apptypes.EventRedelegation{
		Application:      &foundApp,
		SessionEndHeight: sessionEndHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		err = fmt.Errorf("failed to emit application redelegation event: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}
	logger.Info(fmt.Sprintf("Emitted application redelegation event %v", event))

	isSuccessful = true
	return &apptypes.MsgUndelegateFromGatewayResponse{}, nil
}

// recordPendingUndelegation adds the given gateway address to the application's
// pending undelegations list.
func (k Keeper) recordPendingUndelegation(
	ctx context.Context,
	app *apptypes.Application,
	gatewayAddress string,
	currentBlockHeight int64,
) {
	sessionEndHeight := uint64(k.sharedKeeper.GetSessionEndHeight(ctx, currentBlockHeight))
	undelegatingGatewayListAtBlock := app.PendingUndelegations[sessionEndHeight]

	// Add the gateway address to the undelegated gateways list if it's not already there.
	if !slices.Contains(undelegatingGatewayListAtBlock.GatewayAddresses, gatewayAddress) {
		undelegatingGatewayListAtBlock.GatewayAddresses = append(
			undelegatingGatewayListAtBlock.GatewayAddresses,
			gatewayAddress,
		)
		app.PendingUndelegations[sessionEndHeight] = undelegatingGatewayListAtBlock
	} else {
		k.logger.Info(fmt.Sprintf(
			"Application with address [%s] undelegating (again) from a gateway it's already undelegating from with address [%s]",
			app.Address,
			gatewayAddress,
		))
	}

}
