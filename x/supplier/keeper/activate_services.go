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

	// Iterate through all suppliers to check for pending service activations.
	activatedServiceConfigsIterator := k.GetActivatedServiceConfigUpdatesIterator(ctx, currentHeight)
	defer activatedServiceConfigsIterator.Close()

	for ; activatedServiceConfigsIterator.Valid(); activatedServiceConfigsIterator.Next() {
		supplierConfigUpdate, err := activatedServiceConfigsIterator.Value()
		if err != nil {
			logger.Error(fmt.Sprintf("could not get service config update from iterator: %v", err))
			return 0, err
		}

		activatedConfigsSuppliers[supplierConfigUpdate.OperatorAddress] = struct{}{}
	}

	for supplierOperatorAddr := range activatedConfigsSuppliers {
		supplier, found := k.GetSupplier(ctx, supplierOperatorAddr)
		if !found {
			logger.Error(fmt.Sprintf("could not find supplier %s", supplierOperatorAddr))
			continue
		}

		event := &suppliertypes.EventSupplierServiceConfigActivated{
			Supplier:         &supplier,
			ActivationHeight: currentHeight,
		}
		// Emit service activation events.
		if err := suppliertypes.EmitEventSupplierServiceConfigActivated(ctx, event); err != nil {
			logger.Error(fmt.Sprintf("could not emit event %v", event))
			return 0, err
		}
	}

	return len(activatedConfigsSuppliers), nil
}
