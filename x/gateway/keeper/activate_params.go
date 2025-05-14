package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateGatewayParams activates the gateway params that are scheduled
// to be effective at the start of the current session.
func (k Keeper) BeginBlockerActivateGatewayParams(
	ctx context.Context,
) (activatedGatewayParamsUpdate *gatewaytypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateGatewayParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedGatewayParamsUpdate, nil
	}

	for _, gatewayParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if gatewayParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the gateway params.
		k.SetParams(ctx, gatewayParamsUpdate.Params)
		activatedGatewayParamsUpdate = gatewayParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &gatewaytypes.EventParamsActivated{
			ParamsUpdate: *gatewayParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedGatewayParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedGatewayParamsUpdate, nil
}
