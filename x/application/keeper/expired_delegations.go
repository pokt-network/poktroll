package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pokt-network/poktroll/x/application/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// archivedDelegationsRetentionBlocks is the time in terms of blocks number
// for which archived delegations are retained.
// Past this limit, they are pruned from their respective application's records.
// TODO_DISCUSS: Currently, the retention period is in terms of SessionGracePeriod,
// but it could be a governance parameter too.
var archivedDelegationsRetentionBlocks = 3 * sessionkeeper.GetSessionGracePeriodBlockCount()

// EndBlockerPruneExpiredDelegateeSets prunes expired delegations from applications
// at the end of each session by removing archived delegations that are older
// than the retention period.
func (k Keeper) EndBlockerPruneExpiredDelegations(ctx sdk.Context) error {
	logger := k.Logger().With("method", "EndBlockerPruneExpiredDelegations")

	// Since archiving delegations is done when an application is undelegated
	// from a gateway and this only happens at the end of a session, we can prune
	// expired delegatee sets at the end of a session without missing any.
	currentBlockHeight := ctx.BlockHeight()
	sessionEndBlockHeight := sessionkeeper.GetSessionEndBlockHeight(currentBlockHeight)
	if currentBlockHeight != sessionEndBlockHeight {
		return nil
	}

	// Do not prune delegations if archivedDelegationsRetentionBlocks takes
	// it past the first session.
	nextSessionStartHeight := currentBlockHeight + 1
	if nextSessionStartHeight < archivedDelegationsRetentionBlocks {
		return nil
	}

	retentionHeight := nextSessionStartHeight - archivedDelegationsRetentionBlocks

	logger.Info("Pruning expired undelegations")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	archivedDelegationsKeyPrefix := types.KeyPrefix(types.ArchivedDelegationsKeyPrefix)
	store := prefix.NewStore(storeAdapter, archivedDelegationsKeyPrefix)

	// Iterate over the archived delegations store to find referenced applications
	// that have archived delegations to prune.
	// Each entry in the archived delegations store corresponds to a session end
	// block height where old delegations archiving took place (i.e when pending
	// undelegations are processed).
	// Iterate in reverse order to ensure that deletions from the store do not
	// affect the iteration process.
	req := &query.PageRequest{Reverse: true}
	// Iterate over all archived delegations and delete the ones that are older
	// than the retention period.
	for {
		pageRes, err := query.Paginate(store, req, func(key []byte, value []byte) error {
			var appsWithArchivedDelegations types.ApplicationsWithArchivedDelegations
			k.cdc.MustUnmarshal(value, &appsWithArchivedDelegations)
			// Skip if the delegations got archived after the retention height.
			if appsWithArchivedDelegations.LastActiveBlockHeight >= retentionHeight {
				return nil
			}

			// Delete expired delegations from the applications that are referenced
			// by the current archived delegations entry.
			k.deleteExpiredDelegations(
				ctx,
				appsWithArchivedDelegations.AppAddresses,
				retentionHeight,
			)
			// Delete the archived delegations entry.
			store.Delete(key)
			return nil
		})

		if err != nil {
			logger.Error("Error querying archived delegations", err)
			return err
		}

		// Break if there are no more pages to iterate over.
		if pageRes.NextKey == nil {
			break
		}

		// Update the key for the next page.
		req.Key = pageRes.NextKey
	}

	return nil
}

// referenceAppsWithArchivedDelegations creates a reference to the applications
// that have been effectively undelegating in the current session's end block.
// This reference is used to avoid iterating over all applications to find the
// the ones with archived delegations to prune.
func (k Keeper) referenceAppsWithArchivedDelegations(
	ctx sdk.Context,
	appsWithArchivedDelegations *types.ApplicationsWithArchivedDelegations,
) {
	k.logger.Info("Referencing applications having newly archived delegations")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	archivedDelegationsKeyPrefix := types.KeyPrefix(types.ArchivedDelegationsKeyPrefix)
	store := prefix.NewStore(storeAdapter, archivedDelegationsKeyPrefix)

	store.Set(
		types.ArchivedDelegationsBlockKey(appsWithArchivedDelegations.LastActiveBlockHeight),
		k.cdc.MustMarshal(appsWithArchivedDelegations),
	)
}

// deleteExpiredDelegations deletes the archived delegations prior to the retention
// height from the applications with the given addresses.
func (k Keeper) deleteExpiredDelegations(
	ctx sdk.Context,
	appAddresses []string,
	retentionHeight int64,
) {
	logger := k.Logger().With("method", "deleteExpiredDelegations")

	for _, appAddr := range appAddresses {
		// Retrieve the application to delete the expired delegations from.
		foundApp, isAppFound := k.GetApplication(ctx, appAddr)
		if !isAppFound {
			logger.Warn(fmt.Sprintf("Application with address %q not found", appAddr))
		}

		// Iterate over the application's archived delegations and delete the ones
		// that are older than the retention height.
		for i, archivedDelegation := range foundApp.ArchivedDelegations {
			if archivedDelegation.LastActiveBlockHeight >= retentionHeight {
				continue
			}
			logger.Info(fmt.Sprintf(`Deleting expired delegation for appAddress %q"`, appAddr))
			foundApp.ArchivedDelegations = append(
				foundApp.ArchivedDelegations[:i],
				foundApp.ArchivedDelegations[i+1:]...,
			)
		}

		// Update the application with the new archived delegations.
		k.SetApplication(ctx, foundApp)
	}
}
