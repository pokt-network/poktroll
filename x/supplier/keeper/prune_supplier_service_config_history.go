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
//
// IMPORTANT: Service configs are kept until AFTER the proof window closes for the last session
// they could have been used in. This ensures historical session queries return correct supplier
// lists when validating claims.
func (k Keeper) EndBlockerPruneSupplierServiceConfigHistory(
	ctx context.Context,
) (numSuppliersWithPrunedHistory int, err error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	logger := k.Logger().With("method", "PruneSupplierServiceConfigHistory")

	// Get shared params to calculate proof window
	sharedParams := k.sharedKeeper.GetParams(ctx)

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

		// CRITICAL: Only prune configs if we're past the proof window for the last session
		// they could have been active in.
		//
		// The deactivation height marks when the config stopped being active for NEW sessions.
		// However, claims can still be submitted for sessions that ENDED before deactivation.
		// We must keep the config until the proof window closes for those sessions.
		//
		// Calculate: deactivationHeight + proof_window_close_offset
		// This ensures the config is available for historical session queries during claim validation.
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, serviceConfigUpdate.DeactivationHeight)

		if currentHeight <= proofWindowCloseHeight {
			// Config is still needed for claim validation - skip pruning
			logger.Debug(fmt.Sprintf(
				"skipping pruning of service config for supplier %s, service %s: proof window not closed (current: %d, proof window closes: %d)",
				serviceConfigUpdate.OperatorAddress,
				serviceConfigUpdate.Service.ServiceId,
				currentHeight,
				proofWindowCloseHeight,
			))
			continue
		}

		// Safe to delete - proof window has closed
		k.deleteDeactivatedServiceConfigUpdate(ctx, serviceConfigUpdate)

		// Record that this supplier had configurations pruned
		deactivatedConfigsSuppliers[serviceConfigUpdate.OperatorAddress] = true
	}

	return len(deactivatedConfigsSuppliers), nil
}
