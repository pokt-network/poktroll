package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateSharedParams activates the shared params that are scheduled
// to be effective at the start of the current session.
// It returns the params update that was activated.
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

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new shared params",
		sharedtypes.GetSessionNumber(sharedParamsUpdates, currentBlockHeight),
	))

	for _, sharedParamsUpdate := range sharedParamsUpdates {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if sharedParamsUpdate.EffectiveBlockHeight != uint64(currentBlockHeight) {
			continue
		}

		// Effectively update the shared params.
		k.SetParams(ctx, sharedParamsUpdate.Params)
		activatedSharedParamsUpdate = &sharedParamsUpdate

		// Emit params update event.
		eventParamsUpdated := &sharedtypes.EventParamsUpdated{
			Params:               sharedParamsUpdate.Params,
			EffectiveBlockHeight: sharedParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsUpdated))
			return activatedSharedParamsUpdate, err
		}

		break
	}

	return activatedSharedParamsUpdate, nil
}
