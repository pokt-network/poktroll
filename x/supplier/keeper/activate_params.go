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
func (k Keeper) BeginBlockerActivateSupplierParams(
	ctx context.Context,
) (activatedSupplierParamsUpdate *suppliertypes.ParamsUpdate, err error) {
	logger := k.Logger().With("method", "ActivateSupplierParams")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()
	sharedParamsUpdates := k.sharedKeeper.GetParamsUpdates(ctx)

	// Only activate params at the start of a session.
	if !sharedtypes.IsSessionStartHeight(sharedParamsUpdates, currentBlockHeight) {
		return activatedSupplierParamsUpdate, nil
	}

	for _, supplierParamsUpdate := range k.GetParamsUpdates(ctx) {
		// Skip updates that are not scheduled to be effective at the current block height.
		// ActivationHeight is set by UpdateParams and UpdateParam keeper methods
		// to be a session start height.
		if supplierParamsUpdate.ActivationHeight != currentBlockHeight {
			continue
		}

		// Effectively update the supplier params.
		k.SetParams(ctx, supplierParamsUpdate.Params)
		activatedSupplierParamsUpdate = supplierParamsUpdate

		// Emit params update event.
		eventParamsActivated := &suppliertypes.EventParamsActivated{
			ParamsUpdate: *supplierParamsUpdate,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(eventParamsActivated); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", eventParamsActivated))
			return activatedSupplierParamsUpdate, err
		}

		sdkCtx.EventManager().EmitEvent(cosmostypes.NewEvent("pocket.EventParamsActivated"))

		break
	}

	return activatedSupplierParamsUpdate, nil
}
