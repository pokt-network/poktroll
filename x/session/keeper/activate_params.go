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
// It returns the params update that was activated.
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

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new session params",
		sharedtypes.GetSessionNumber(sharedParamsUpdates, currentBlockHeight),
	))

	for _, sessionParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if sessionParamsUpdate.EffectiveBlockHeight != uint64(currentBlockHeight) {
			continue
		}

		// Effectively update the shared params.
		k.SetParams(ctx, sessionParamsUpdate.Params)
		activatedSessionParamsUpdate = sessionParamsUpdate

		// Emit params update event.
		eventParamsUpdated := &sessiontypes.EventParamsUpdated{
			Params:               sessionParamsUpdate.Params,
			EffectiveBlockHeight: sessionParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsUpdated))
			return activatedSessionParamsUpdate, err
		}

		break
	}

	return activatedSessionParamsUpdate, nil
}
