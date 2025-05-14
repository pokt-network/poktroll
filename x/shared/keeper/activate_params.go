package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateSharedParams activates the shared params that are scheduled
// to be effective at the start of the current session.
func (k Keeper) BeginBlockerActivateSharedParams(
	ctx context.Context,
) (activatedSharedParamsUpdate *sharedtypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateSharedParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedSharedParamsUpdate, nil
	}

	for _, sharedParamsUpdate := range sharedParamsUpdates {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if sharedParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the shared params.
		k.SetParams(ctx, sharedParamsUpdate.Params)
		activatedSharedParamsUpdate = sharedParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &sharedtypes.EventParamsActivated{
			ParamsUpdate: *sharedParamsUpdate,
		}

		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedSharedParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedSharedParamsUpdate, nil
}
