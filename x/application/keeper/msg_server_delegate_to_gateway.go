package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) DelegateToGateway(ctx context.Context, msg *apptypes.MsgDelegateToGateway) (*apptypes.MsgDelegateToGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"delegate_to_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "DelegateToGateway")
	logger.Info(fmt.Sprintf("About to delegate application to gateway with msg: %+v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Delegation Message failed basic validation: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Retrieve the application from the store
	app, found := k.GetApplication(ctx, msg.GetAppAddress())
	if !found {
		logger.Info(fmt.Sprintf("Application not found with address [%s]", msg.AppAddress))
		return nil, status.Error(
			codes.NotFound,
			apptypes.ErrAppNotFound.Wrapf(
				"application address: %s", msg.GetAppAddress(),
			).Error(),
		)
	}
	logger.Info(fmt.Sprintf("Application found with address [%s]", msg.AppAddress))

	// Check if the gateway is staked
	if _, found := k.gatewayKeeper.GetGateway(ctx, msg.GetGatewayAddress()); !found {
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
				"application already delegated to %d (max) gateways", maxDelegatedParam,
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

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info(fmt.Sprintf("Successfully delegated application to gateway for app: %+v", app))

	// Emit the application redelegation event
	currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	event := &apptypes.EventRedelegation{
		Application:      &app,
		SessionEndHeight: sessionEndHeight,
	}
	logger.Info(fmt.Sprintf("Emitting application redelegation event %+v", event))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &apptypes.MsgDelegateToGatewayResponse{
		Application: &app,
	}, nil
}
