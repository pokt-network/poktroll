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
// It returns the params update that was activated.
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

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new tokenomics params",
		sharedtypes.GetSessionNumber(sharedParamsUpdates, currentBlockHeight),
	))

	for _, tokenomicsParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if tokenomicsParamsUpdate.EffectiveBlockHeight != uint64(currentBlockHeight) {
			continue
		}

		// Effectively update the tokenomics params.
		k.SetParams(ctx, tokenomicsParamsUpdate.Params)
		activatedTokenomicsParamsUpdate = tokenomicsParamsUpdate

		// Emit a params update event.
		eventParamsUpdated := &tokenomicstypes.EventParamsUpdated{
			Params:               tokenomicsParamsUpdate.Params,
			EffectiveBlockHeight: tokenomicsParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsUpdated))
			return activatedTokenomicsParamsUpdate, err
		}

		break
	}

	return activatedTokenomicsParamsUpdate, nil
}
