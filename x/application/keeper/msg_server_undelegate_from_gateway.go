package keeper

import (
	"context"
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

func (k msgServer) UndelegateFromGateway(ctx context.Context, msg *types.MsgUndelegateFromGateway) (*types.MsgUndelegateFromGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"undelegate_from_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UndelegateFromGateway")
	logger.Info(fmt.Sprintf("About to undelegate application from gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Undelegation Message failed basic validation: %v", err))
		return nil, err
	}

	// Retrieve the application from the store
	foundApp, isAppFound := k.GetApplication(ctx, msg.AppAddress)
	if !isAppFound {
		logger.Info(fmt.Sprintf("Application not found with address [%s]", msg.AppAddress))
		return nil, types.ErrAppNotFound.Wrapf("application not found with address: %s", msg.AppAddress)
	}
	logger.Info(fmt.Sprintf("Application found with address [%s]", msg.AppAddress))

	// Check if the application is already delegated to the gateway
	foundIdx := -1
	for i, gatewayAddr := range foundApp.DelegateeGatewayAddresses {
		if gatewayAddr == msg.GatewayAddress {
			foundIdx = i
		}
	}
	if foundIdx == -1 {
		logger.Info(fmt.Sprintf("Application not delegated to gateway with address [%s]", msg.GatewayAddress))
		return nil, types.ErrAppNotDelegated.Wrapf("application not delegated to gateway with address: %s", msg.GatewayAddress)
	}

	// Remove the gateway from the application's delegatee gateway public keys
	foundApp.DelegateeGatewayAddresses = append(foundApp.DelegateeGatewayAddresses[:foundIdx], foundApp.DelegateeGatewayAddresses[foundIdx+1:]...)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := sdkCtx.BlockHeight()
	sessionEndHeight := uint64(sessionkeeper.GetSessionEndBlockHeight(currentBlock))

	// TODO_INVESTIGATE: When there is no undelegation, the undelegations map is nil
	// even though it was declared with nullable=false in the application proto.
	if foundApp.Undelegations == nil {
		foundApp.Undelegations = make(map[uint64]types.UndelegationFromAppToGatewayEvent)
	}

	// Create a session undelegation entry for the given session end height if it doesn't exist.
	undelegationsAtBlock, ok := foundApp.Undelegations[sessionEndHeight]
	if !ok {
		undelegationsAtBlock = types.UndelegationFromAppToGatewayEvent{
			UndelegatedGateways: []string{},
		}
	}

	// Add the gateway to the undelegated gateways list if it's not already there.
	if !slices.Contains(undelegationsAtBlock.UndelegatedGateways, msg.GatewayAddress) {
		undelegationsAtBlock.UndelegatedGateways = append(
			undelegationsAtBlock.UndelegatedGateways,
			msg.GatewayAddress,
		)

		foundApp.Undelegations[sessionEndHeight] = undelegationsAtBlock
	}

	// Update the application store with the new delegation
	k.SetApplication(ctx, foundApp)
	logger.Info(fmt.Sprintf("Successfully undelegated application from gateway for app: %+v", foundApp))

	// Emit the application redelegation event
	event := msg.NewRedelegationEvent()
	logger.Info(fmt.Sprintf("Emitting application redelegation event %v", event))

	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %v", err))
		return nil, err
	}

	isSuccessful = true
	return &types.MsgUndelegateFromGatewayResponse{}, nil
}
