package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// EndBlockerPruneSupplierServiceConfigHistory prunes the service config history of existing suppliers.
// If a supplier has updated its supported set of configs, but that history is no longer needed
// for various reasons (servicing relays, claim settlement, etc), it can be pruned.
// This helps reduce onchain state bloat and avoid diverting attention from non-actionable metadata.
func (k Keeper) EndBlockerPruneSupplierServiceConfigHistory(
	ctx context.Context,
) (numSuppliersWithPrunedHistory int, err error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	logger := k.Logger().With("method", "PruneSupplierServiceConfigHistory")

	// Track unique suppliers whose configurations were pruned
	deactivatedConfigsSuppliers := make(map[string]bool)

	// Retrieve all service configurations that should be deactivated at the current height
	deactivatedServiceConfigsIterator := k.GetDeactivatedServiceConfigUpdatesIterator(ctx, currentHeight)
	defer deactivatedServiceConfigsIterator.Close()

	for ; deactivatedServiceConfigsIterator.Valid(); deactivatedServiceConfigsIterator.Next() {
		serviceConfigUpdate, err := deactivatedServiceConfigsIterator.Value()
		if err != nil {
			logger.Error(fmt.Sprintf("could not get service config update from iterator: %v", err))
			return 0, err
		}

		// Delete the deactivated service config and all its indexes
		k.deleteDeactivatedServiceConfigUpdate(ctx, serviceConfigUpdate)

		// Record that this supplier had configurations pruned
		deactivatedConfigsSuppliers[serviceConfigUpdate.OperatorAddress] = true
	}

	return len(deactivatedConfigsSuppliers), nil
}
