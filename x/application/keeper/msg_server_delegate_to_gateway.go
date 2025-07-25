package keeper

import (
	"context"
	"fmt"
	"slices"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// DelegateToGateway processes a message to delegate an application to a gateway.
// This enables the application to use the gateway's services for relaying.
func (k msgServer) DelegateToGateway(ctx context.Context, msg *apptypes.MsgDelegateToGateway) (*apptypes.MsgDelegateToGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"delegate_to_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	logger := k.Logger().With("method", "DelegateToGateway")
	logger.Info(fmt.Sprintf("About to delegate application to gateway with msg: %+v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Info(fmt.Sprintf("Delegation Message failed basic validation: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Retrieve the application from the store
	app, found := k.GetApplication(ctx, msg.GetAppAddress())
	if !found {
		logger.Info(fmt.Sprintf("Application not found with address [%s]", msg.AppAddress))
		return nil, status.Error(
			codes.NotFound,
			apptypes.ErrAppNotFound.Wrapf(
				"application with address: %s", msg.GetAppAddress(),
			).Error(),
		)
	}
	logger.Info(fmt.Sprintf("Application found with address [%s]", msg.AppAddress))

	// Check if the gateway is staked
	gateway, gatewayFound := k.gatewayKeeper.GetGateway(ctx, msg.GetGatewayAddress())
	if !gatewayFound {
		logger.Info(fmt.Sprintf("Gateway not found with address [%s]", msg.GetGatewayAddress()))
		return nil, status.Error(
			codes.NotFound,
			apptypes.ErrAppGatewayNotFound.Wrapf(
				"gateway with address: %q", msg.GetGatewayAddress(),
			).Error(),
		)
	}

	// Ensure the application is not already delegated to the maximum number of gateways
	maxDelegatedParam := k.GetParams(ctx).MaxDelegatedGateways
	if uint64(len(app.DelegateeGatewayAddresses)) >= maxDelegatedParam {
		logger.Info(fmt.Sprintf("Application already delegated to maximum number of gateways: %d", maxDelegatedParam))
		return nil, status.Error(
			codes.FailedPrecondition,
			apptypes.ErrAppMaxDelegatedGateways.Wrapf(
				"application with address %q already delegated to %d (max) gateways",
				msg.GetAppAddress(), maxDelegatedParam,
			).Error(),
		)
	}

	currentHeight := sdkCtx.BlockHeight()
	// Ensure that the gateway is still active
	if !gateway.IsActive(currentHeight) {
		logger.Info(fmt.Sprintf("Gateway with address [%s] is unbonding and no longer active", msg.GetGatewayAddress()))
		return nil, status.Error(
			codes.FailedPrecondition,
			gatewaytypes.ErrGatewayIsInactive.Wrapf(
				"gateway with address: %q", msg.GetGatewayAddress(),
			).Error(),
		)
	}

	// Check if the application is already delegated to the gateway
	for _, gatewayAddr := range app.DelegateeGatewayAddresses {
		if gatewayAddr == msg.GetGatewayAddress() {
			logger.Info(fmt.Sprintf("Application already delegated to gateway with address [%s]", msg.GatewayAddress))
			return nil, status.Error(
				codes.AlreadyExists,
				apptypes.ErrAppAlreadyDelegated.Wrapf(
					"application with address %q already delegated to gateway with address: %q",
					msg.GetAppAddress(), msg.GetGatewayAddress(),
				).Error(),
			)
		}
	}

	// Update the application with the new delegatee public key
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, msg.GetGatewayAddress())
	logger.Info("Successfully added delegatee public key to application")

	// Remove any pending undelegations for the application to the gateway
	k.updatePendingUndelegations(ctx, &app, msg.GetGatewayAddress(), logger)

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info(fmt.Sprintf("Successfully delegated application to gateway for app: %+v", app))

	// Emit the application redelegation event
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	event := &apptypes.EventRedelegation{
		Application:      &app,
		SessionEndHeight: sessionEndHeight,
	}
	logger.Info(fmt.Sprintf("Emitting application redelegation event %+v", event))

	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		err = fmt.Errorf("failed to emit application redelegation event: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &apptypes.MsgDelegateToGatewayResponse{}, nil
}

// updatePendingUndelegations removes the given gateway address from the application's
// pending undelegations list.
func (k Keeper) updatePendingUndelegations(
	ctx context.Context,
	app *apptypes.Application,
	gatewayAddress string,
	logger log.Logger,
) {
	// Check if the application has any pending undelegations
	if len(app.PendingUndelegations) == 0 {
		return
	}

	for height, pendingUndelegations := range app.PendingUndelegations {
		if gwIdx := slices.Index(pendingUndelegations.GatewayAddresses, gatewayAddress); gwIdx >= 0 {
			// Remove the gateway address from the pending undelegations
			pendingUndelegations.GatewayAddresses = append(
				pendingUndelegations.GatewayAddresses[:gwIdx],
				pendingUndelegations.GatewayAddresses[gwIdx+1:]...,
			)
			logger.Info(fmt.Sprintf("Removed pending undelegation for re-delegated gateway with address [%s]", gatewayAddress))
		}
		app.PendingUndelegations[height] = pendingUndelegations

		if len(app.PendingUndelegations[height].GatewayAddresses) == 0 {
			// If there are no more pending undelegations for this height, remove it from the application
			delete(app.PendingUndelegations, height)
		}
	}
}
