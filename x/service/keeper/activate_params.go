package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateServiceParams activates the service params that are scheduled
// to be effective at the start of the current session.
// It returns the params update that was activated.
func (k Keeper) BeginBlockerActivateServiceParams(
	ctx context.Context,
) (activatedServiceParamsUpdate *servicetypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateServiceParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedServiceParamsUpdate, nil
	}

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new service params",
		sharedtypes.GetSessionNumber(sharedParamsUpdates, currentBlockHeight),
	))

	for _, serviceParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if serviceParamsUpdate.EffectiveBlockHeight != uint64(currentBlockHeight) {
			continue
		}

		// Effectively update the service params.
		k.SetParams(ctx, serviceParamsUpdate.Params)
		activatedServiceParamsUpdate = serviceParamsUpdate

		// Emit params update event.
		eventParamsUpdated := &servicetypes.EventParamsUpdated{
			Params:               serviceParamsUpdate.Params,
			EffectiveBlockHeight: serviceParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsUpdated))
			return activatedServiceParamsUpdate, err
		}

		break
	}

	return activatedServiceParamsUpdate, nil
}
