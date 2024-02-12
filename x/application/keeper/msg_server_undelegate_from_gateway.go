package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UndelegateFromGateway(goCtx context.Context, msg *types.MsgUndelegateFromGateway) (*types.MsgUndelegateFromGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "UndelegateFromGateway")
	logger.Info(fmt.Sprintf("About to undelegate application from gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Undelegation Message failed basic validation: %v", err))
		return nil, err
	}

	// Retrieve the application from the store
	app, found := k.GetApplication(ctx, msg.AppAddress)
	if !found {
		logger.Info(fmt.Sprintf("Application not found with address [%s]", msg.AppAddress))
		return nil, sdkerrors.Wrapf(types.ErrAppNotFound, "application not found with address: %s", msg.AppAddress)
	}
	logger.Info(fmt.Sprintf("Application found with address [%s]", msg.AppAddress))

	// Check if the application is already delegated to the gateway
	foundIdx := -1
	for i, gatewayAddr := range app.DelegateeGatewayAddresses {
		if gatewayAddr == msg.GatewayAddress {
			foundIdx = i
		}
	}
	if foundIdx == -1 {
		logger.Info(fmt.Sprintf("Application not delegated to gateway with address [%s]", msg.GatewayAddress))
		return nil, sdkerrors.Wrapf(types.ErrAppNotDelegated, "application not delegated to gateway with address: %s", msg.GatewayAddress)
	}

	// Remove the gateway from the application's delegatee gateway public keys
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses[:foundIdx], app.DelegateeGatewayAddresses[foundIdx+1:]...)

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info(fmt.Sprintf("Successfully undelegated application from gateway for app: %+v", app))

	// Emit the application redelegation event
	event := msg.NewRedelegationEvent()
	logger.Info(fmt.Sprintf("Emitting application redelegation event %v", event))
	if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
		logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %v", err))
		return nil, err
	}

	return &types.MsgUndelegateFromGatewayResponse{}, nil
}
