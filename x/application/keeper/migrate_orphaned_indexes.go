package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	"github.com/pokt-network/poktroll/x/application/types"
)

// CleanupOrphanedUndelegationIndexes removes orphaned undelegation index entries
// that reference applications which no longer exist in the store.
//
// This can occur when an application with pending undelegations is removed
// (unstaked or transferred) but the undelegation index entries are not properly
// cleaned up due to a bug in removeApplicationUndelegationIndex that was
// deleting from the delegation store instead of the undelegation store.
//
// Returns the count of orphaned entries removed and any error.
func (k Keeper) CleanupOrphanedUndelegationIndexes(ctx context.Context) (int, error) {
	undelegationStore := k.getUndelegationStore(ctx)
	applicationStore := k.getApplicationStore(ctx)

	// Build set of valid application keys
	validAppKeys := make(map[string]bool)
	appIter := storetypes.KVStorePrefixIterator(applicationStore, []byte{})
	defer appIter.Close()

	for ; appIter.Valid(); appIter.Next() {
		validAppKeys[string(appIter.Key())] = true
	}

	// Iterate all undelegation index entries and collect orphaned ones
	toDelete := make([][]byte, 0)
	undelegationIter := storetypes.KVStorePrefixIterator(undelegationStore, []byte{})
	defer undelegationIter.Close()

	for ; undelegationIter.Valid(); undelegationIter.Next() {
		var undelegation types.PendingUndelegation
		k.cdc.MustUnmarshal(undelegationIter.Value(), &undelegation)

		appKey := types.ApplicationKey(undelegation.ApplicationAddress)
		if !validAppKeys[string(appKey)] {
			toDelete = append(toDelete, undelegationIter.Key())
		}
	}

	// Delete all orphaned entries
	for _, key := range toDelete {
		undelegationStore.Delete(key)
	}

	return len(toDelete), nil
}
