package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UndelegateFromGateway(
	goCtx context.Context,
	msg *types.MsgUndelegateFromGateway,
) (*types.MsgUndelegateFromGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "UndelegateFromGateway")
	logger.Info("About to undelegate application from gateway with msg: %v", msg)

	if err := msg.ValidateBasic(); err != nil {
		logger.Error("Undelegation Message failed basic validation: %v", err)
		return nil, err
	}

	// Retrieve the application from the store
	app, found := k.GetApplication(ctx, msg.AppAddress)
	if !found {
		logger.Info("Application not found with address [%s]", msg.AppAddress)
		return nil, sdkerrors.Wrapf(types.ErrAppNotFound, "application not found with address: %s", msg.AppAddress)
	}
	logger.Info("Application found with address [%s]", msg.AppAddress)

	// Check if the application is already delegated to the gateway
	foundIdx := -1
	for i, gatewayAddr := range app.DelegateeGatewayAddresses {
		if gatewayAddr == msg.GatewayAddress {
			foundIdx = i
		}
	}
	if foundIdx == -1 {
		logger.Info("Application not delegated to gateway with address [%s]", msg.GatewayAddress)
		return nil, sdkerrors.Wrapf(types.ErrAppNotDelegated, "application not delegated to gateway with address: %s", msg.GatewayAddress)
	}

	// Remove the gateway from the application's delegatee gateway public keys
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses[:foundIdx], app.DelegateeGatewayAddresses[foundIdx+1:]...)

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info("Successfully undelegated application from gateway for app: %+v", app)

	// Emit the application delegation change event
	ctx.EventManager().EmitTypedEvent(msg.NewDelegateeChangeEvent())

	return &types.MsgUndelegateFromGatewayResponse{}, nil
}
