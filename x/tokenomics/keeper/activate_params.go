package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// BeginBlockerActivateTokenomicsParams activates the tokenomics params that are
// scheduled to be effective at the start of the current session.
func (k Keeper) BeginBlockerActivateTokenomicsParams(
	ctx context.Context,
) (activatedTokenomicsParamsUpdate *tokenomicstypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateTokenomicsParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedTokenomicsParamsUpdate, nil
	}

	for _, tokenomicsParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if tokenomicsParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the tokenomics params.
		k.SetParams(ctx, tokenomicsParamsUpdate.Params)
		activatedTokenomicsParamsUpdate = tokenomicsParamsUpdate

		// Emit params activation event.
		eventParamsActivated := &tokenomicstypes.EventParamsActivated{
			ParamsUpdate: *tokenomicsParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedTokenomicsParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedTokenomicsParamsUpdate, nil
}
