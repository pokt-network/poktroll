package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) DelegateToGateway(ctx context.Context, msg *types.MsgDelegateToGateway) (*types.MsgDelegateToGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"delegate_to_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "DelegateToGateway")
	logger.Info(fmt.Sprintf("About to delegate application to gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Delegation Message failed basic validation: %v", err))
		return nil, err
	}

	// Retrieve the application from the store
	app, found := k.GetApplication(ctx, msg.AppAddress)
	if !found {
		logger.Info(fmt.Sprintf("Application not found with address [%s]", msg.AppAddress))
		return nil, types.ErrAppNotFound.Wrapf("application not found with address: %s", msg.AppAddress)
	}
	logger.Info(fmt.Sprintf("Application found with address [%s]", msg.AppAddress))

	// Check if the gateway is staked
	if _, found := k.gatewayKeeper.GetGateway(ctx, msg.GatewayAddress); !found {
		logger.Info(fmt.Sprintf("Gateway not found with address [%s]", msg.GatewayAddress))
		return nil, types.ErrAppGatewayNotFound.Wrapf("gateway not found with address: %s", msg.GatewayAddress)
	}

	// Ensure the application is not already delegated to the maximum number of gateways
	maxDelegatedParam := k.GetParams(ctx).MaxDelegatedGateways
	if uint64(len(app.DelegateeGatewayAddresses)) >= maxDelegatedParam {
		logger.Info(fmt.Sprintf("Application already delegated to maximum number of gateways: %d", maxDelegatedParam))
		return nil, types.ErrAppMaxDelegatedGateways.Wrapf("application already delegated to %d gateways", maxDelegatedParam)
	}

	// Check if the application is already delegated to the gateway
	for _, gatewayAddr := range app.DelegateeGatewayAddresses {
		if gatewayAddr == msg.GatewayAddress {
			logger.Info(fmt.Sprintf("Application already delegated to gateway with address [%s]", msg.GatewayAddress))
			return nil, types.ErrAppAlreadyDelegated.Wrapf("application already delegated to gateway with address: %s", msg.GatewayAddress)
		}
	}

	// Update the application with the new delegatee public key
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, msg.GatewayAddress)
	logger.Info(fmt.Sprintf("Successfully added delegatee public key to application"))

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)
	logger.Info(fmt.Sprintf("Successfully delegated application to gateway for app: %+v", app))

	// Emit the application redelegation event
	event := msg.NewRedelegationEvent()
	logger.Info(fmt.Sprintf("Emitting application redelegation event %v", event))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %v", err))
		return nil, err
	}

	isSuccessful = true
	return &types.MsgDelegateToGatewayResponse{}, nil
}
