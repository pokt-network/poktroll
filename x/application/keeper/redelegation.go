package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/slices"

	"github.com/pokt-network/poktroll/x/application/types"
)

// SetRedelegation sets the Relelegation in two stores, one indexed by the
// app & gateway address, the other by the entries completion block height.
func (k Keeper) SetRedelegation(
	ctx sdk.Context,
	redelegation types.Redelegation,
) error {
	// Create stores for both the redelegation indexed by its ID and by its
	// completion block height.
	redelegationStore := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.RedelegationPrimaryKeyPrefix),
	)
	completionStore := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.RedelegationCompletionPrimaryKeyPrefix),
	)

	// Serialize the redelegation
	b := k.cdc.MustMarshal(&redelegation)

	// Attempt to retrieve the redelegations remaining in the store
	redelegations, found := k.GetRedelegation(
		ctx,
		redelegation.AppAddress,
		redelegation.GatewayAddress,
	)
	var sortedEntries []types.RedelegationEntry
	if found {
		// If incomplete redelegations exist, sort their entries by their
		// completion height, so they can be reinserted in the correct order.
		entries := append(redelegation.Entries, redelegations.Entries...)
		slices.SortFunc[[]types.RedelegationEntry, types.RedelegationEntry](
			entries,
			func(a, b types.RedelegationEntry) int {
				if a.CompletionHeight < b.CompletionHeight {
					return -1
				}
				if a.CompletionHeight > b.CompletionHeight {
					return 1
				}
				return 0
			},
		)
		sortedEntries = entries
	}

	// Store the redelegation in the store according to the entries ID.
	redelegationStore.Set(types.RedelegationPrimaryKey(
		redelegation.AppAddress,
		redelegation.GatewayAddress,
	), b)

	// Sort the entries by their completion block height if none have been found.
	if !found {
		slices.SortFunc(redelegation.Entries, func(a, b types.RedelegationEntry) int {
			if a.CompletionHeight < b.CompletionHeight {
				return -1
			}
			if a.CompletionHeight > b.CompletionHeight {
				return 1
			}
			return 0
		})
		sortedEntries = redelegation.Entries
	}

	// For each remaining entry set it in the completion store, indexed by the
	// completion block height, the block height it was made at and it's ID.
	for _, entry := range sortedEntries {
		b := k.cdc.MustMarshal(&entry)
		completionStore.Set(types.RedelegationCompletionPrimaryKey(
			entry.CompletionHeight,
			entry.CurrentHeight,
			entry.RedelegationId,
		), b)
	}

	return nil
}

// GetRedelegation returns a redelegation instance from the app & gateway
// addresses provided. It contains all redelegation entries yet to complete.
func (k Keeper) GetRedelegation(
	ctx sdk.Context,
	appAddress, gatewayAddress string,
) (re types.Redelegation, found bool) {
	// Create the store to retrieve redelegations by app & gateway address.
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.RedelegationPrimaryKeyPrefix),
	)

	// Retrieve the redelegation from the store
	b := store.Get(types.RedelegationPrimaryKey(
		appAddress,
		gatewayAddress,
	))
	// If no redelegation exists, return an empty redelegation instance and false.
	if b == nil {
		return re, false
	}

	// Unmarshal the redelegation and return it
	k.cdc.MustUnmarshal(b, &re)
	return re, true
}
