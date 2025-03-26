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
// It returns the params update that was activated.
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

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new gateway params",
		sharedtypes.GetSessionNumber(sharedParamsUpdates, currentBlockHeight),
	))

	for _, gatewayParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if gatewayParamsUpdate.EffectiveBlockHeight != uint64(currentBlockHeight) {
			continue
		}

		// Effectively update the gateway params.
		k.SetParams(ctx, gatewayParamsUpdate.Params)
		activatedGatewayParamsUpdate = gatewayParamsUpdate

		// Emit params update event.
		eventParamsUpdated := &gatewaytypes.EventParamsUpdated{
			Params:               gatewayParamsUpdate.Params,
			EffectiveBlockHeight: gatewayParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsUpdated))
			return activatedGatewayParamsUpdate, err
		}

		break
	}

	return activatedGatewayParamsUpdate, nil
}
