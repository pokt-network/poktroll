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

	for _, serviceParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if serviceParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the service params.
		k.SetParams(ctx, serviceParamsUpdate.Params)
		activatedServiceParamsUpdate = serviceParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &servicetypes.EventParamsActivated{
			ParamsUpdate: *serviceParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedServiceParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedServiceParamsUpdate, nil
}
