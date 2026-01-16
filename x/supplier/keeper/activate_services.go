package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// BeginBlockerActivateSupplierServices processes suppliers that have pending service activations
// at the current block height.
// It returns the number of suppliers whose services were activated.
func (k Keeper) BeginBlockerActivateSupplierServices(
	ctx context.Context,
) (numSuppliersWithServicesActivation int, err error) {
	logger := k.Logger().With("method", "ActivateSupplierServices")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only activate supplier services at the start of a session.
	if !sharedtypes.IsSessionStartHeight(&sharedParams, currentHeight) {
		return numSuppliersWithServicesActivation, nil
	}

	logger.Info(fmt.Sprintf(
		"starting session %d, about to activate services for suppliers",
		sharedtypes.GetSessionNumber(&sharedParams, currentHeight),
	))

	activatedConfigsSuppliers := make(map[string]struct{})

	// Iterate through all service config updates to check for pending service activations.
	activatedServiceConfigsIterator := k.GetActivatedServiceConfigUpdatesIterator(ctx, currentHeight)
	defer activatedServiceConfigsIterator.Close()

	// TODO_IMPROVE: With some tweaks to the activatedServiceConfigsIterator, we may be able to
	// emit a single EventSupplierServiceConfigActivated with a repeated service_ids field.
	// This would minimize the onchain disk utilization of the event.
	for ; activatedServiceConfigsIterator.Valid(); activatedServiceConfigsIterator.Next() {
		supplierConfigUpdate, err := activatedServiceConfigsIterator.Value()
		if err != nil {
			// Log and skip orphaned index entries instead of failing
			// This handles cases where index entries point to deleted primary records
			logger.Debug(fmt.Sprintf("skipping orphaned service config index entry: %v", err))
			continue
		}

		activatedConfigsSuppliers[supplierConfigUpdate.OperatorAddress] = struct{}{}

		// Emit event for each activated service configuration
		event := &suppliertypes.EventSupplierServiceConfigActivated{
			OperatorAddress:  supplierConfigUpdate.OperatorAddress,
			ServiceId:        supplierConfigUpdate.Service.ServiceId,
			ActivationHeight: currentHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", event))
			return 0, err
		}
	}

	return len(activatedConfigsSuppliers), nil
}
