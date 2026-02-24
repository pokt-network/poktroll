package keeper

// ┌───────────────────────────────────────────────────────────────────────────────────────────────┐
// │ 🗺️  Application Index Map                                                                     │
// ├───────────────────────────────────────────────────────────────────────────────────────────────┤
// │ Store (bucket)                              Key                              → Value          │
// │───────────────────────────────────────────────────────────────────────────────────────────────│
// │ applicationUnstakingStore                   AK                               → AK             │
// │ applicationTransferStore                    AK                               → AK             │
// │ delegationStore                             DK (GatewayAddr || AppAddr)      → AK             │
// │ undelegationStore                           UK (AppAddr   || GatewayAddr)    → undelegationBz │
// └───────────────────────────────────────────────────────────────────────────────────────────────┘
//
// Legend
//   ||                  : byte-level concatenation / prefix.
//   AK (ApplicationKey) : types.ApplicationKey(appAddr) = "Application/address/" || appAddr.
//   DK (DelegationKey)  : types.DelegationKey(gatewayAddr, appAddr)
//                         = "Application/delegation/"   || gatewayAddr || appAddr.
//   UK (UndelegationKey): types.UndelegationKey(appAddr, gatewayAddr)
//                         = "Application/undelegation/" || appAddr     || gatewayAddr.
//   undelegationBz       : protobuf-marshaled types.PendingUndelegation.
//
// Fast-path look-ups
//   • Unstaking set           → iterate applicationUnstakingStore keys.          (①)
//   • Pending transfers       → iterate applicationTransferStore keys.           (②)
//   • Delegated apps (by GW)  → delegationStore prefix-scan GatewayAddr.         (③)
//   • Pending undelegations   → undelegationStore prefix-scan AppAddr/Gateway.   (④)
//
// Index counts
//   ① Unstaking applications
//   ② Applications with pending transfers
//   ③ Application ↔ Gateway delegations
//   ④ Pending undelegations

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

// Maintains an index of applications with a pending transfer to another address.
//
// Behavior:
// - Adds an application to the transfer index if it has a pending transfer
// - Removes an application from the index if there is no pending transfer
//
// Purpose:
// - Enables tracking of applications in the process of transferring their stake to another address.
func (k Keeper) indexApplicationTransfer(ctx context.Context, app types.Application) {
	appTransferStore := k.getApplicationTransferStore(ctx)

	appKey := types.ApplicationKey(app.Address)
	if app.HasPendingTransfer() {
		appTransferStore.Set(appKey, appKey)
	} else {
		appTransferStore.Delete(appKey)
	}
}

// Maintains an index of which applications are delegated to which gateways.
//
// Behavior:
// - Recreates the delegation index for an application
// - Establishes relationship links between the application and all its delegated gateways
//
// Index structure:
// - Keys combine gateway and application addresses
// - Allows efficient lookups for which applications are delegated to a given gateway.
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

// Maintains an index of pending undelegations for applications from gateways.
//
// Behavior:
// 1. Deletes any existing undelegation indexes for the application
// 2. Recreates the undelegation index based on current pending undelegations
//
// Purpose:
// - Enables efficient tracking of:
//   - Which gateways are being undelegated from by a specific application
//   - All pending undelegations across the network.
func (k Keeper) indexApplicationUndelegations(ctx context.Context, app types.Application) {
	appUndelegationStore := k.getUndelegationStore(ctx)
	appDelegationStore := k.getDelegationStore(ctx)

	// Delete and recreate all undelegations indexes of the given application
	appUndelegationsIterator := k.GetUndelegationsIterator(ctx, app.Address)
	defer appUndelegationsIterator.Close()

	// First, remove all existing undelegation indexes for this application
	undelegationsKeys := make([][]byte, 0)
	for ; appUndelegationsIterator.Valid(); appUndelegationsIterator.Next() {
		undelegationsKeys = append(undelegationsKeys, appUndelegationsIterator.Key())
	}
	for _, undelegationKey := range undelegationsKeys {
		appUndelegationStore.Delete(undelegationKey)
	}

	// Recreate the undelegation index for the application by adding
	// entries for each pending undelegation at each undelegation height
	for _, undelegationsAtHeight := range app.PendingUndelegations {
		for _, undelegatedGatewayAddress := range undelegationsAtHeight.GatewayAddresses {
			undelegationKey := types.UndelegationKey(app.Address, undelegatedGatewayAddress)
			undelegation := &types.PendingUndelegation{
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

// Removes an application from the unstaking index.
//
// Usage:
// - Call when an application is fully removed or completes the unstaking process.
func (k Keeper) removeApplicationUnstakingIndex(
	ctx context.Context,
	applicationAddress string,
) {
	appUnstakingStore := k.getApplicationUnstakingStore(ctx)
	appKey := types.ApplicationKey(applicationAddress)
	appUnstakingStore.Delete(appKey)
}

// Removes an application from the transfer index.
//
// Usage:
// - Call when an application completes or cancels a pending transfer.
func (k Keeper) removeApplicationTransferIndex(
	ctx context.Context,
	applicationAddress string,
) {
	appTransferStore := k.getApplicationTransferStore(ctx)
	appKey := types.ApplicationKey(applicationAddress)
	appTransferStore.Delete(appKey)
}

// Removes a specific application-gateway delegation relationship from the delegation index.
//
// Usage:
// - Call when an application undelegates from a specific gateway.
func (k Keeper) removeApplicationDelegationIndex(
	ctx context.Context,
	applicationAddress string,
	gatewayAddress string,
) {
	appDelegationStore := k.getDelegationStore(ctx)
	delegationKey := types.DelegationKey(gatewayAddress, applicationAddress)
	appDelegationStore.Delete(delegationKey)
}

// Removes all delegation indexes for a specific application.
//
// Usage:
// - Call when cleaning up an application's data (e.g. fully unstaked or transferred).
func (k Keeper) removeApplicationDelegationsIndexes(
	ctx context.Context,
	application types.Application,
) {
	for _, gatewayAddress := range application.DelegateeGatewayAddresses {
		k.removeApplicationDelegationIndex(ctx, application.Address, gatewayAddress)
	}
}

// Removes all undelegation indexes for a specific application.
//
// Usage:
// - Call when cleaning up an application's data (e.g. fully unstaked or transferred)
// - Ensures no orphaned undelegation records remain in the store.
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

// Removes a specific undelegation record from the undelegation index using its key.
func (k Keeper) removeApplicationUndelegationIndex(
	ctx context.Context,
	undelegationKey []byte,
) {
	appUndelegationStore := k.getUndelegationStore(ctx)
	appUndelegationStore.Delete(undelegationKey)
}
