package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BeginBlockerActivateProofParams activates the proof params that are scheduled
// to be effective at the start of the current session.
func (k Keeper) BeginBlockerActivateProofParams(
	ctx context.Context,
) (activatedProofParamsUpdate *prooftypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateProofParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedProofParamsUpdate, nil
	}

	for _, proofParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if proofParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the proof params.
		k.SetParams(ctx, proofParamsUpdate.Params)
		activatedProofParamsUpdate = proofParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &prooftypes.EventParamsActivated{
			ParamsUpdate: *proofParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedProofParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedProofParamsUpdate, nil
}
