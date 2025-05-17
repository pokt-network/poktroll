package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateApplicationParams activates the application params that are scheduled
// to be effective at the start of the current session.
func (k Keeper) BeginBlockerActivateApplicationParams(
	ctx context.Context,
) (activatedApplicationParamsUpdate *apptypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateApplicationParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedApplicationParamsUpdate, nil
	}

	for _, applicationParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if applicationParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the application params.
		k.SetParams(ctx, applicationParamsUpdate.Params)
		activatedApplicationParamsUpdate = applicationParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &apptypes.EventParamsActivated{
			ParamsUpdate: *applicationParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedApplicationParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedApplicationParamsUpdate, nil
}
