package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/application/types"
)

const ALL_UNDELEGATIONS = ""

// indexApplicationUnstaking maintains an index of applications that are
// currently in the unbonding period.
//
// This function either adds or removes an application from the unstaking index
// depending on whether the application is currently unbonding:
// - If the application is unbonding (UnstakeSessionEndHeight > 0), it's added to the index
// - If the application is not unbonding, it's removed from the index
//
// DEV_NOTE: Since an unstaking application cannot perform successive unbonding
// actions until it re-stakes or completes the unbonding period, we can safely
// use the application's address as the key in the index.
//
// TODO_IMPROVE: Consider having an unbondingHeight/address key for even more efficient lookups.
// This involves processing the unbonding height here in addition to the unbonding EndBlocker.
//
// This index enables the EndBlocker to efficiently find applications that are unbonding
// without iterating over and unmarshaling all applications in the store.
func (k Keeper) indexApplicationUnstaking(ctx context.Context, app types.Application) {
	appUnstakingStore := k.getApplicationUnstakingStore(ctx)

	appKey := types.ApplicationKey(app.Address)
	if app.IsUnbonding() {
		appUnstakingStore.Set(appKey, appKey)
	} else {
		appUnstakingStore.Delete(appKey)
	}
}

// indexApplicationTransfer maintains an index of applications that have a
// pending transfer to another address.
//
// This function either adds or removes an application from the transfer index
// depending on whether the application has a pending transfer:
// - If the application has a pending transfer, it's added to the index
// - If the application doesn't have a pending transfer, it's removed from the index
//
// This index enables tracking of applications that are in the process of transferring
// their stake to another address.
func (k Keeper) indexApplicationTransfer(ctx context.Context, app types.Application) {
	appTransferStore := k.getApplicationTransferStore(ctx)

	appKey := types.ApplicationKey(app.Address)
	if app.HasPendingTransfer() {
		appTransferStore.Set(appKey, appKey)
	} else {
		appTransferStore.Delete(appKey)
	}
}

// indexApplicationDelegations maintains an index of which applications are delegated
// to which gateways.
//
// This function recreates the delegation index for an application, establishing
// relationship links between the application and all its delegated gateways.
//
// The index is structured with keys that combine gateway and application addresses,
// allowing efficient lookups for which applications are delegated to a given gateway.
func (k Keeper) indexApplicationDelegations(ctx context.Context, app types.Application) {
	appDelegationStore := k.getDelegationStore(ctx)

	// Pending undelegations list contains gateways that were previously delegated
	// to but are now being removed.
	// Clean up any delegation index records for these gateways.
	for _, undelegationsAtHeight := range app.PendingUndelegations {
		for _, undelegatedGatewayAddress := range undelegationsAtHeight.GatewayAddresses {
			delegationKey := types.DelegationKey(undelegatedGatewayAddress, app.Address)
			appDelegationStore.Delete(delegationKey)
		}
	}

	// Recreate the delegation index for the application
	for _, delegatedGatewayAddress := range app.DelegateeGatewayAddresses {
		delegationKey := types.DelegationKey(delegatedGatewayAddress, app.Address)
		applicationKey := types.ApplicationKey(app.Address)
		appDelegationStore.Set(delegationKey, applicationKey)
	}
}

// indexApplicationUndelegations maintains an index of pending undelegations
// for applications from gateways.
//
// This function manages the storage of undelegation records:
// 1. First deletes any existing undelegation indexes for the application
// 2. Then recreates the undelegation index based on the current pending undelegations
//
// The undelegation index allows efficient tracking of:
// - Which gateways are being undelegated from by a specific application
// - All pending undelegations across the network
func (k Keeper) indexApplicationUndelegations(ctx context.Context, app types.Application) {
	appUndelegationStore := k.getUndelegationStore(ctx)
	appDelegationStore := k.getDelegationStore(ctx)

	// Delete and recreate all undelegations indexes of the given application
	appUndelegationsIterator := k.GetUndelegationsIterator(ctx, app.Address)
	defer appUndelegationsIterator.Close()

	// First, remove all existing undelegation indexes for this application
	undelegationsKyes := make([][]byte, 0)
	for ; appUndelegationsIterator.Valid(); appUndelegationsIterator.Next() {
		undelegationsKyes = append(undelegationsKyes, appUndelegationsIterator.Key())
	}
	for _, undelegationKey := range undelegationsKyes {
		appUndelegationStore.Delete(undelegationKey)
	}

	// Recreate the undelegation index for the application by adding
	// entries for each pending undelegation at each undelegation height
	for _, undelegationsAtHeight := range app.PendingUndelegations {
		for _, undelegatedGatewayAddress := range undelegationsAtHeight.GatewayAddresses {
			undelegationKey := types.UndelegationKey(app.Address, undelegatedGatewayAddress)
			undelegation := &types.Undelegation{
				ApplicationAddress: app.Address,
				GatewayAddress:     undelegatedGatewayAddress,
			}
			undelegationBz := k.cdc.MustMarshal(undelegation)
			appUndelegationStore.Set(undelegationKey, undelegationBz)

			// Remove any indexed delegation record corresponding to this undelegation
			delegationKey := types.DelegationKey(undelegatedGatewayAddress, app.Address)
			appDelegationStore.Delete(delegationKey)
		}
	}
}

// removeApplicationUnstakingIndex removes an application from the unstaking index.
//
// Used when an application is fully removed or completes the unstaking process.
func (k Keeper) removeApplicationUnstakingIndex(
	ctx context.Context,
	applicationAddress string,
) {
	appUnstakingStore := k.getApplicationUnstakingStore(ctx)
	appKey := types.ApplicationKey(applicationAddress)
	appUnstakingStore.Delete(appKey)
}

// removeApplicationTransferIndex removes an application from the transfer index.
//
// Used when an application completes or cancels a pending transfer.
func (k Keeper) removeApplicationTransferIndex(
	ctx context.Context,
	applicationAddress string,
) {
	appTransferStore := k.getApplicationTransferStore(ctx)
	appKey := types.ApplicationKey(applicationAddress)
	appTransferStore.Delete(appKey)
}

// removeApplicationDelegationIndex removes a specific application-gateway delegation relationship
// from the delegation index.
//
// Called when an application undelegates from a specific gateway.
func (k Keeper) removeApplicationDelegationIndex(
	ctx context.Context,
	applicationAddress string,
	gatewayAddress string,
) {
	appDelegationStore := k.getDelegationStore(ctx)
	delegationKey := types.DelegationKey(gatewayAddress, applicationAddress)
	appDelegationStore.Delete(delegationKey)
}

// removeApplicationUndelegationIndexes removes all undelegation indexes for a specific application.
//
// Used when cleaning up an application's data, such as when it's fully unstaked or transferred.
// This ensures no orphaned undelegation records remain in the store.
func (k Keeper) removeApplicationUndelegationIndexes(
	ctx context.Context,
	applicationAddress string,
) {
	// Get all undelegations for this application
	appUndelegationsIterator := k.GetUndelegationsIterator(ctx, applicationAddress)
	defer appUndelegationsIterator.Close()

	// Collect undelegation keys to avoid iterator invalidation during deletion
	appUndelegations := make([][]byte, 0)
	for ; appUndelegationsIterator.Valid(); appUndelegationsIterator.Next() {
		appUndelegations = append(appUndelegations, appUndelegationsIterator.Key())
	}

	// Delete each undelegation index
	for _, undelegationKey := range appUndelegations {
		k.removeApplicationUndelegationIndex(ctx, undelegationKey)
	}
}

// removeApplicationUndelegationIndex removes a specific undelegation record
// from the undelegation index using its key.
func (k Keeper) removeApplicationUndelegationIndex(
	ctx context.Context,
	undelegationKey []byte,
) {
	appDelegationStore := k.getDelegationStore(ctx)
	appDelegationStore.Delete(undelegationKey)
}
