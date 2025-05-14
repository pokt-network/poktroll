package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateSessionParams activates the session params that are scheduled
// to be effective at the start of the current session.
func (k Keeper) BeginBlockerActivateSessionParams(
	ctx context.Context,
) (activatedSessionParamsUpdate *sessiontypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateSessionParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedSessionParamsUpdate, nil
	}

	for _, sessionParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if sessionParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the shared params.
		k.SetParams(ctx, sessionParamsUpdate.Params)
		activatedSessionParamsUpdate = sessionParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &sessiontypes.EventParamsActivated{
			ParamsUpdate: *sessionParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedSessionParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedSessionParamsUpdate, nil
}
