package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k Keeper) BeginBlockerActivateSupplierServices(
	ctx context.Context,
) (numSuppliersWithServicesActivation uint64, err error) {
	logger := k.Logger().With("method", "ActivateSupplierServices")

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only activate supplier services at the start of a session.
	if sharedtypes.IsSessionStartHeight(&sharedParams, currentHeight) {
		return numSuppliersWithServicesActivation, nil
	}

	// Iterate over all suppliers and activate their services if the current height
	// is equal to the supplier's service activation height.
	// TODO_POST_MAINNET: Use an index to iterate over suppliers that have services
	// to activate at the current height.
	for _, supplier := range k.GetAllSuppliers(ctx) {
		lastUpdateIdx := len(supplier.ServicesUpdateHistory) - 1
		// Skip suppliers that have no services to activate.
		// supplier.ServicesUpdateHistory might be empty if the history has been pruned.
		if lastUpdateIdx < 0 {
			continue
		}

		if supplier.ServicesUpdateHistory[lastUpdateIdx].UpdateHeight == uint64(currentHeight) {
			logger.Info(
				"Activating services for supplier",
				"operator_address", supplier.OperatorAddress,
				"services", supplier.ServicesUpdateHistory[lastUpdateIdx].Services,
			)

			// Update the supplier's services.
			supplier.Services = supplier.ServicesUpdateHistory[lastUpdateIdx].Services

			// Save the updated supplier.
			k.SetSupplier(ctx, supplier)
			numSuppliersWithServicesActivation += 1
		}
	}

	return numSuppliersWithServicesActivation, nil
}
