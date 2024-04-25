package keeper

import (
	"context"
	"fmt"
	"slices"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/pokt-network/poktroll/x/application/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// EndBlockerProcessPendingUndelegations processes undelegations that are
// scheduled to be effective at the beginning of the next session.
// It updates the applications that are undelegating from gateways and removes
// the undelegations from the pending undelegations store.
// It also updates the applications' archived delegations by appending the
// previously active ones to the archived delegations.
// Applications that have their archived delegations updated are referenced
// in the ApplicationsWithArchivedDelegations store.
func (k Keeper) EndBlockerProcessPendingUndelegations(ctx sdk.Context) error {
	logger := k.Logger().With("method", "EndBlockerProcessPendingUndelegations")

	currentBlockHeight := ctx.BlockHeight()
	sessionEndBlockHeight := sessionkeeper.GetSessionEndBlockHeight(currentBlockHeight)
	if currentBlockHeight != sessionEndBlockHeight {
		return nil
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	undelegationKeyPrefix := types.KeyPrefix(types.PendingUndelegationsKeyPrefix)
	store := prefix.NewStore(storeAdapter, undelegationKeyPrefix)

	logger.Info(fmt.Sprintf(
		"Processing pending undelegations to be effective at block %d",
		sessionEndBlockHeight+1,
	))

	// Collect all pending undelegations in a map of application address to
	// undelegation slices.
	appsPendingUndelegations := map[string][]*types.Undelegation{}
	req := &query.PageRequest{}
	// Iterate over all the pages of pending undelegations store.
	for {
		pageRes, err := query.Paginate(store, req, func(key []byte, value []byte) error {
			var undelegation types.Undelegation
			k.cdc.MustUnmarshal(value, &undelegation)
			// Create a new entry in the map if the application is not already present.
			if _, ok := appsPendingUndelegations[undelegation.AppAddress]; !ok {
				appsPendingUndelegations[undelegation.AppAddress] = []*types.Undelegation{}
			}
			// Append the undelegation to the application's slice.
			appsPendingUndelegations[undelegation.AppAddress] = append(
				appsPendingUndelegations[undelegation.AppAddress],
				&undelegation,
			)
			return nil
		})

		if err != nil {
			logger.Error("Error querying pending undelegations", err)
			return err
		}

		// Break if there are no more pages to query.
		if pageRes.NextKey == nil {
			break
		}

		// Update the request key to query the next page.
		req.Key = pageRes.NextKey
	}

	// Prepare the applications that need to be referenced in the sessionEndBlockHeight
	// archived delegations.
	appsWithArchivedDelegations := &types.ApplicationsWithArchivedDelegations{
		LastActiveBlockHeight: sessionEndBlockHeight,
	}

	// Iterate over the applications with pending undelegations and process them
	// by effectively undelegating from the gateways.
	for appAddr, undelegations := range appsPendingUndelegations {
		if err := k.undelegateFromGateways(ctx, appAddr, undelegations, sessionEndBlockHeight); err != nil {
			logger.Error("Error undelegating from gateway", err)
			return err
		}
		appsWithArchivedDelegations.AppAddresses = append(
			appsWithArchivedDelegations.AppAddresses,
			appAddr,
		)
	}

	// Any application that got undelegated from gateways in the current session's
	// end block will have its previous delegations archived and its address
	// referenced in the ApplicationsWithArchivedDelegations store.
	if len(appsPendingUndelegations) > 0 {
		k.referenceAppsWithArchivedDelegations(ctx, appsWithArchivedDelegations)

		logger.Info(fmt.Sprintf(
			"Processed pending undelegations for %d applications",
			len(appsPendingUndelegations),
		))
	}

	return nil
}

// undelegateFromGateways effectively undelegates the application from the
// gateways, updates the application's delegatee gateway addresses and updates
// the application's archived delegations.
func (k Keeper) undelegateFromGateways(
	ctx sdk.Context,
	appAddr string,
	undelegations []*types.Undelegation,
	lastActiveBlockHeight int64,
) error {
	logger := k.Logger().With("method", "undelegateFromGateway")

	// Retrieve the application from the store
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	if !isAppFound {
		return types.ErrAppNotFound.Wrapf("application not found with address %q", appAddr)
	}

	// Update the application's archived delegations before undelegating.
	archivedDelegations := types.ArchivedDelegations{
		LastActiveBlockHeight: lastActiveBlockHeight,
		GatewayAddresses:      foundApp.DelegateeGatewayAddresses,
	}
	foundApp.ArchivedDelegations = append(foundApp.ArchivedDelegations, archivedDelegations)

	for _, undelegation := range undelegations {
		// Check if the application is already delegated to the gateway
		gatewayIdx := slices.Index(foundApp.DelegateeGatewayAddresses, undelegation.GatewayAddress)
		if gatewayIdx == -1 {
			return types.ErrAppNotDelegated.Wrapf(
				"application not delegated to gateway with address %q",
				undelegation.GatewayAddress,
			)
		}

		// Remove the gateway from the application's delegatee gateway public keys
		foundApp.DelegateeGatewayAddresses = append(
			foundApp.DelegateeGatewayAddresses[:gatewayIdx],
			foundApp.DelegateeGatewayAddresses[gatewayIdx+1:]...,
		)

		// Remove the undelegation from the pending undelegations store
		k.removePendingUndelegation(ctx, undelegation)

		logger.Info(fmt.Sprintf(
			"Application with address %q undelegated from gateway with address %q",
			undelegation.AppAddress,
			undelegation.GatewayAddress,
		))

		logger.Info(fmt.Sprintf("Emitting application redelegation event %v", undelegation))

		// Emit the application redelegation event
		redelegationEvent := &types.EventRedelegation{
			AppAddress:     undelegation.AppAddress,
			GatewayAddress: undelegation.GatewayAddress,
		}
		if err := ctx.EventManager().EmitTypedEvent(redelegationEvent); err != nil {
			logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %v", err))
			return err
		}
	}

	// Update the application store with the new delegation
	k.SetApplication(ctx, foundApp)

	return nil
}

// removePendingUndelegation removes the undelegation from the pending
// undelegations store.
func (k Keeper) removePendingUndelegation(
	ctx context.Context,
	pendingUndelegation *types.Undelegation,
) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(
		storeAdapter,
		types.KeyPrefix(types.PendingUndelegationsKeyPrefix),
	)

	hasPendingUndelegation := store.Has(types.PendingUndelegationKey(pendingUndelegation))
	if hasPendingUndelegation {
		k.Logger().With("method", "removePendingUndelegation").
			Info(fmt.Sprintf("Removing from pending undelegations %v", pendingUndelegation))

		store.Delete(types.PendingUndelegationKey(pendingUndelegation))
	}
}
