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

	// Basic validation of the message
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

	// TODO_INVESTIGATE: When there are no undelegatiosn, the undelegations map is nil
	// even though it was declared with nullable=false in the application proto.
	if foundApp.PendingUndelegations == nil {
		foundApp.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
	}

	// Create a session undelegation list entry for the given session end height if it doesn't already exist.
	undelegatingGatewayListAtBlock, ok := foundApp.PendingUndelegations[sessionEndHeight]
	if !ok {
		undelegatingGatewayListAtBlock = types.UndelegatingGatewayList{
			GatewayAddresses: []string{},
		}
	}

	// Add the gateway address to the list undelegated gateways list if it's not already there.
	if !slices.Contains(undelegatingGatewayListAtBlock.GatewayAddresses, msg.GatewayAddress) {
		undelegatingGatewayListAtBlock.GatewayAddresses = append(
			undelegatingGatewayListAtBlock.GatewayAddresses,
			msg.GatewayAddress,
		)
		foundApp.PendingUndelegations[sessionEndHeight] = undelegatingGatewayListAtBlock
	} else {
		logger.Warn(fmt.Sprintf("Application undelegating (again) from gateway it's already undelegating from with address [%s]", msg.GatewayAddress))

	}

	// Update the application store with the new delegation
	k.SetApplication(ctx, foundApp)
	logger.Info(fmt.Sprintf("Successfully undelegated application from gateway for app: %+v", foundApp))

	// Emit the application redelegation event
	event := msg.NewRedelegationEvent()
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %v", err))
		return nil, err
	}
	logger.Info(fmt.Sprintf("Emmited application redelegation event %v", event))

	isSuccessful = true
	return &types.MsgUndelegateFromGatewayResponse{}, nil
}
