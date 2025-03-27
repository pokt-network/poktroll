package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// BeginBlockerActivateSupplierParams activates the supplier params that are
// scheduled to be effective at the start of the current session.
// It returns the params update that was activated.
func (k Keeper) BeginBlockerActivateSupplierParams(
	ctx context.Context,
) (activatedSupplierParamsUpdate *suppliertypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateSupplierParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(&sharedParamsUpdates, currentBlockHeight) {
		return activatedSupplierParamsUpdate, nil
	}

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate new supplier params",
		sharedtypes.GetSessionNumber(&sharedParamsUpdates, currentBlockHeight),
	))

	for _, supplierParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// EffectiveBlockHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if supplierParamsUpdate.EffectiveBlockHeight != uint64(currentBlockHeight) {
			continue
		}

		// Effectively update the shared params.
		k.SetParams(ctx, supplierParamsUpdate.Params)
		activatedSupplierParamsUpdate = &supplierParamsUpdate

		// Emit params update event.
		eventParamsUpdated := &suppliertypes.EventParamsUpdated{
			Params:               supplierParamsUpdate.Params,
			EffectiveBlockHeight: supplierParamsUpdate.EffectiveBlockHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsUpdated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsUpdated))
			return activatedSupplierParamsUpdate, err
		}

		break
	}

	return activatedSupplierParamsUpdate, nil
}
