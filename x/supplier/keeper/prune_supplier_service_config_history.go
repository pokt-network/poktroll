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
// IMPORTANT: Service configs are kept until AFTER the latest proof window close of the sessions
// they could have been used in. This ensures historical session queries return correct supplier
// lists when validating claims.
func (k Keeper) EndBlockerPruneSupplierServiceConfigHistory(
	ctx context.Context,
) (numSuppliersWithPrunedHistory int, err error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	logger := k.Logger().With("method", "PruneSupplierServiceConfigHistory")

	// Track unique suppliers whose configurations were pruned
	deactivatedConfigsSuppliers := make(map[string]bool)

	// Many configs share activation and deactivation heights: compute each
	// proof window close height once per block.
	proofWindowCloseHeights := make(map[[2]int64]int64)

	// Retrieve all service configurations that were deactivated at or before the current height.
	// We use a range iterator because configs may not be pruned at their exact deactivation height
	// (the proof window typically hasn't closed yet), and we need to find them later when the
	// proof window has closed.
	deactivatedServiceConfigsIterator := k.GetDeactivatedServiceConfigUpdatesRangeIterator(ctx, currentHeight)
	defer deactivatedServiceConfigsIterator.Close()

	for ; deactivatedServiceConfigsIterator.Valid(); deactivatedServiceConfigsIterator.Next() {
		serviceConfigUpdate, err := deactivatedServiceConfigsIterator.Value()
		if err != nil {
			// Log and skip orphaned index entries instead of failing
			// This handles cases where index entries point to deleted primary records
			logger.Debug(fmt.Sprintf("skipping orphaned service config index entry: %v", err))
			continue
		}

		// CRITICAL: Only prune configs if we're past the proof window for the last session
		// they could have been active in.
		//
		// The deactivation height marks when the config stopped being active for NEW sessions.
		// However, claims can still be submitted for sessions that ENDED before deactivation.
		// We must keep the config until the proof window closes for those sessions.
		//
		// The deactivation height is set to the FIRST block of the next session (see
		// msg_server_unstake_supplier.go). Therefore, the LAST block where the config
		// was active is DeactivationHeight - 1, which falls within the last active session.
		// We keep the config until the latest proof window close of the sessions it
		// was active in, which is usually that last session's.
		lastActiveHeight := serviceConfigUpdate.DeactivationHeight - 1
		heights := [2]int64{serviceConfigUpdate.ActivationHeight, lastActiveHeight}
		proofWindowCloseHeight, ok := proofWindowCloseHeights[heights]
		if !ok {
			proofWindowCloseHeight = k.lastProofWindowCloseHeight(ctx, heights[0], heights[1])
			proofWindowCloseHeights[heights] = proofWindowCloseHeight
		}

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

// lastProofWindowCloseHeight returns the latest proof window close height of the
// sessions between activationHeight and lastActiveHeight. The last session
// usually closes last, but an earlier session closes later when a params change
// since then shortened the claim and proof windows.
func (k Keeper) lastProofWindowCloseHeight(ctx context.Context, activationHeight, lastActiveHeight int64) int64 {
	sharedParams := k.sharedKeeper.GetParamsAtHeight(ctx, lastActiveHeight)
	closeHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, lastActiveHeight)

	// Each history entry starts a params epoch: the session containing the block
	// before it is the last one under the previous params.
	k.sharedKeeper.IterateParamsHistoryReverse(ctx, lastActiveHeight, func(effectiveHeight int64, _ sharedtypes.Params) bool {
		if effectiveHeight <= activationHeight {
			return true
		}
		previousParams := k.sharedKeeper.GetParamsAtHeight(ctx, effectiveHeight-1)
		if previousParams.GetNumBlocksPerSession() == 0 {
			return false
		}
		closeHeight = max(closeHeight, sharedtypes.GetProofWindowCloseHeight(&previousParams, effectiveHeight-1))
		return false
	})

	return closeHeight
}
