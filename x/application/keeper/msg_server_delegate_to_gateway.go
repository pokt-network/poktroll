package keeper

import (
	"context"

	"pocket/x/application/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) DelegateToGateway(goCtx context.Context, msg *types.MsgDelegateToGateway) (*types.MsgDelegateToGatewayResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "DelegateToGateway")
	logger.Info("About to delegate application to gateway with msg: %v", msg)

	if err := msg.ValidateBasic(); err != nil {
		logger.Error("Delegation Message failed basic validation: %v", err)
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
	gateway, found := k.gatewayKeeper.GetGateway(ctx, msg.GatewayAddress)
	if !found {
		logger.Info("Gateway not found with address [%s]", msg.GatewayAddress)
		return nil, sdkerrors.Wrapf(types.ErrAppGatewayNotFound, "gateway not found with address: %s", msg.GatewayAddress)
	}

	// Ensure the application is not already delegated to the maximum number of gateways
	maxDelegatedParam := k.GetParams(ctx).MaxDelegatedGateways
	if int64(len(app.DelegateeGatewayAddresses)) >= maxDelegatedParam {
		logger.Info("Application already delegated to maximum number of gateways: %d", maxDelegatedParam)
		return nil, sdkerrors.Wrapf(types.ErrAppMaxDelegatedGateways, "application already delegated to maximum number of gateways: %d", maxDelegatedParam)
	}

	// Check if the application is already delegated to the gateway
	for _, gatewayAddress := range app.DelegateeGatewayAddresses {
		if gatewayAddress == msg.GatewayAddress {
			logger.Info("Application already delegated to gateway with address: %s", msg.GatewayAddress)
			return nil, sdkerrors.Wrapf(types.ErrAppAlreadyDelegated, "application already delegated to gateway with address: %s", msg.GatewayAddress)
		}
	}

	// Update the application with the new delegatee public key
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, msg.GatewayAddress)
	logger.Info("Successfully added delegatee public key to application")

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info("Successfully delegated application to gateway for app: %+v", app)

	// Update the gateway with the new delegator address
	gateway.DelegatorApplicationAddresses = append(gateway.DelegatorApplicationAddresses, msg.AppAddress)
	logger.Info("Successfully added delegator address to gateway")

	// Update the gateway store with the new delegator address
	k.gatewayKeeper.SetGateway(ctx, gateway)
	logger.Info("Successfully added delegator address to gateway for gateway: %+v", gateway)

	return &types.MsgDelegateToGatewayResponse{}, nil
}
