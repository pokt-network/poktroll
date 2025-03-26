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
	// Get the shared params prior to the activation to determine the session start height.
	// If the NumBlocksPerSession will be updated, it will determine the session start height
	// of the future sessions only.
	previousSharedParams := k.sharedKeeper.GetParams(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(&previousSharedParams, currentBlockHeight) {
		return &sharedtypes.ParamsUpdate, nil
	}

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new shared params",
		sharedtypes.GetSessionNumber(&previousSharedParams, currentBlockHeight),
	))

	for _, sharedParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if sharedParamsUpdate.EffectiveBlockHeight != currentBlockHeight {
			continue
		}

		// Update the shared params.
		k.SetParams(ctx, sharedParamsUpdate.Params)
		activatedSharedParamsUpdate = &sharedParamsUpdate

		// Emit service activation events.
		eventSharedParamsUpdated := &sharedtypes.EventSharedParamsUpdated{
			Params:               sharedParamsUpdate.Params,
			EffectiveBlockHeight: sharedParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventSharedParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventSharedParamsUpdated))
			return activatedSharedParamsUpdate, err
		}

		break
	}

	return activatedSharedParamsUpdate, nil
}
