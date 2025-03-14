package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// EndBlockerPruneSupplierServiceConfigHistory prunes the service config history of existing suppliers.
// If a supplier has updated its supported set of configs, but that history is no longer needed
// for various reasons (servicing relays, claim settlement, etc), it can be pruned.
// This helps reduce onchain state bloat and avoid diverting attention from non-actionable metadata.
func (k Keeper) EndBlockerPruneSupplierServiceConfigHistory(
	ctx context.Context,
) (numSuppliersWithPrunedHistory uint64, err error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdkCtx.BlockHeight()
	// The number of blocks from the end of a session to the end of the proof window close.
	// It is needed to determine how long to retain service config updates for pending claims settlement.
	sessionEndToProofWindowCloseNumBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)

	logger := k.Logger().With("method", "PruneSupplierServiceConfigHistory")

	for _, supplier := range k.GetAllSuppliers(ctx) {
		// Store the original number of historical service configs.
		originalHistoryLength := len(supplier.ServiceConfigHistory)

		// Initialize a slice to retain service config updates that are still needed
		// for pending claims settlement.
		retainedServiceConfigs := make([]*sharedtypes.ServiceConfigUpdate, 0)

		// Iterate through each service config update to check if it is still be needed.
		for _, configUpdate := range supplier.ServiceConfigHistory {
			// Calculate the block height when the session corresponding to this service config update ends.
			sessionEndBlockHeight := sharedtypes.GetSessionEndHeight(&sharedParams, int64(configUpdate.EffectiveBlockHeight))

			// Calculate the final block height until which this config update needs to be retained.
			// This includes the proof window close period after the session ends.
			configRetentionBlockHeight := sessionEndBlockHeight + sessionEndToProofWindowCloseNumBlocks

			// Keep the config update if we haven't passed its retention period.
			if currentHeight <= configRetentionBlockHeight {
				retainedServiceConfigs = append(retainedServiceConfigs, configUpdate)
			}
		}

		// Skip if no pruning is needed (all configs are still needed).
		if len(retainedServiceConfigs) == originalHistoryLength {
			continue
		}

		// Special case: if all service historical service config updates would be pruned,
		// retain the most recent one.
		// This is necessary for the session hydration process that relies on the
		// service config history to determine the current active service configuration.
		if len(retainedServiceConfigs) == 0 {
			retainedServiceConfigs = supplier.ServiceConfigHistory[:1]
		}

		// Update the supplier's service config history with the pruned list.
		supplier.ServiceConfigHistory = retainedServiceConfigs

		k.SetSupplier(ctx, supplier)
		logger.Info(fmt.Sprintf(
			"pruned %d out of %d service config history entries for supplier %s",
			originalHistoryLength-len(retainedServiceConfigs),
			originalHistoryLength,
			supplier.OperatorAddress,
		))

		numSuppliersWithPrunedHistory += 1
	}

	return numSuppliersWithPrunedHistory, nil
}
