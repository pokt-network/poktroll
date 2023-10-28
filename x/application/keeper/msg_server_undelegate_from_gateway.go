package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"pocket/x/application/types"
)

func (k msgServer) UndelegateFromGateway(goCtx context.Context, msg *types.MsgUndelegateFromGateway) (*types.MsgUndelegateFromGatewayResponse, error) {
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

	// Check if the gateway is staked
	// TODO(@h5law): Look into using addresses instead of public keys
	if _, found := k.gatewayKeeper.GetGateway(ctx, msg.GatewayAddress); !found {
		logger.Info("Gateway not found with address [%s]", msg.GatewayAddress)
		return nil, sdkerrors.Wrapf(types.ErrAppGatewayNotFound, "gateway not found with address: %s", msg.GatewayAddress)
	}

	// Check if the application is already delegated to the gateway
	foundIdx := -1
	for i, gatewayPubKey := range app.DelegateeGatewayPubKeys {
		// Convert the any type to a public key
		gatewayPubKey, err := types.AnyToPubKey(gatewayPubKey)
		if err != nil {
			logger.Error("unable to convert any type to public key: %v", err)
			return nil, sdkerrors.Wrapf(types.ErrAppAnyConversion, "unable to convert any type to public key: %v", err)
		}
		// Convert the public key to an address
		gatewayAddress := types.PublicKeyToAddress(gatewayPubKey)
		if gatewayAddress == msg.GatewayAddress {
			foundIdx = i
		}
	}
	if foundIdx == -1 {
		logger.Info("Application not delegated to gateway with address [%s]", msg.GatewayAddress)
		return nil, sdkerrors.Wrapf(types.ErrAppNotDelegated, "application not delegated to gateway with address: %s", msg.GatewayAddress)
	}

	// Remove the gateway from the application's delegatee gateway public keys
	app.DelegateeGatewayPubKeys = append(app.DelegateeGatewayPubKeys[:foundIdx], app.DelegateeGatewayPubKeys[foundIdx+1:]...)

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info("Successfully undelegated application from gateway for app: %+v", app)

	return &types.MsgUndelegateFromGatewayResponse{}, nil
}
