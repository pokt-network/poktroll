package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
)

// CleanupOrphanedServiceConfigIndexes removes orphaned index entries that point
// to non-existent primary store records. This occurs when index entries remain
// after their corresponding primary records were deleted.
//
// Returns counts of cleaned entries: (activation, deactivation, supplier, error)
//
// This function is intended to be called during upgrade handlers to clean up
// historical orphaned entries that accumulated before defensive fixes were added.
func (k Keeper) CleanupOrphanedServiceConfigIndexes(ctx context.Context) (int, int, int, error) {
	// Get all stores
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)
	activationHeightStore := k.getServiceConfigUpdateActivationHeightStore(ctx)
	deactivationHeightStore := k.getServiceConfigUpdateDeactivationHeightStore(ctx)
	supplierServiceConfigUpdateStore := k.getSupplierServiceConfigUpdatesStore(ctx)

	// Build set of valid primary keys by iterating through the primary store
	validKeys := make(map[string]bool)
	primaryIter := serviceConfigUpdateStore.Iterator(nil, nil)
	defer primaryIter.Close()

	for ; primaryIter.Valid(); primaryIter.Next() {
		validKeys[string(primaryIter.Key())] = true
	}

	// Clean each index store and count orphaned entries removed
	actCount := k.cleanOrphanedIndex(activationHeightStore, validKeys)
	deactCount := k.cleanOrphanedIndex(deactivationHeightStore, validKeys)
	supplierCount := k.cleanOrphanedIndex(supplierServiceConfigUpdateStore, validKeys)

	return actCount, deactCount, supplierCount, nil
}

// cleanOrphanedIndex removes orphaned entries from a given index store.
// An entry is orphaned if its value (primary key reference) doesn't exist
// in the validKeys map. Returns the count of orphaned entries removed.
func (k Keeper) cleanOrphanedIndex(indexStore storetypes.KVStore, validKeys map[string]bool) int {
	count := 0
	toDelete := [][]byte{}

	// Collect all orphaned index keys
	iter := indexStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		primaryKey := iter.Value()
		if !validKeys[string(primaryKey)] {
			// This index entry points to a non-existent primary record
			toDelete = append(toDelete, iter.Key())
			count++
		}
	}

	// Delete all orphaned entries
	for _, key := range toDelete {
		indexStore.Delete(key)
	}

	return count
}
