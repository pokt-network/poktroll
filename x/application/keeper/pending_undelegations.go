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

func (k Keeper) EndBlockerProcessPendingUndelegations(ctx sdk.Context) error {
	logger := k.Logger().With("method", "EndBlockerProcessPendingUndelegations")

	currentBlockHeight := ctx.BlockHeight()
	if currentBlockHeight != sessionkeeper.GetSessionEndBlockHeight(currentBlockHeight) {
		return nil
	}
	endingSessionNumber := sessionkeeper.GetSessionNumber(currentBlockHeight)

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	undelegationKeyPrefix := types.KeyPrefix(types.PendingUndelegationsKeyPrefix)
	store := prefix.NewStore(storeAdapter, undelegationKeyPrefix)

	logger.Info("Querying pending undelegations")
	appsPendingDelegations := map[string][]*types.Undelegation{}
	req := &query.PageRequest{}
	for {
		pageRes, err := query.Paginate(store, req, func(key []byte, value []byte) error {
			var undelegation types.Undelegation
			k.cdc.MustUnmarshal(value, &undelegation)
			if _, ok := appsPendingDelegations[undelegation.AppAddress]; !ok {
				appsPendingDelegations[undelegation.AppAddress] = []*types.Undelegation{}
			}
			appsPendingDelegations[undelegation.AppAddress] = append(
				appsPendingDelegations[undelegation.AppAddress],
				&undelegation,
			)
			return nil
		})

		if err != nil {
			logger.Error("Error querying pending undelegations", err)
			return err
		}

		if pageRes.NextKey == nil {
			break
		}

		req.Key = pageRes.NextKey
	}

	appsWithArchivedDelegations := &types.ApplicationsWithArchivedDelegations{}
	for appAddr, undelegations := range appsPendingDelegations {
		if err := k.undelegateFromGateways(ctx, appAddr, undelegations, endingSessionNumber); err != nil {
			logger.Error("Error undelegating from gateway", err)
			return err
		}
		appsWithArchivedDelegations.AppAddresses = append(
			appsWithArchivedDelegations.AppAddresses,
			appAddr,
		)
	}

	k.indexArchivedDelegations(ctx, endingSessionNumber, appsWithArchivedDelegations)

	logger.Info("Successfully processed pending undelegations")

	return nil
}

func (k Keeper) undelegateFromGateways(
	ctx sdk.Context,
	appAddr string,
	undelegations []*types.Undelegation,
	sessionNumber int64,
) error {
	logger := k.Logger().With("method", "undelegateFromGateway")

	// Retrieve the application from the store
	foundApp, isAppFound := k.GetApplication(ctx, appAddr)
	if !isAppFound {
		appNotFoundFmt := "Application not found with address %q"
		logger.Info(fmt.Sprintf(appNotFoundFmt, appAddr))
		return types.ErrAppNotFound.Wrapf(appNotFoundFmt, appAddr)
	}

	logger.Info(fmt.Sprintf("Application found with address %q", appAddr))

	archivedDelegation := types.ArchivedDelegation{
		SessionNumber:             uint64(sessionNumber),
		DelegateeGatewayAddresses: foundApp.DelegateeGatewayAddresses,
	}

	foundApp.ArchivedDelegations = append(foundApp.ArchivedDelegations, archivedDelegation)

	for _, undelegation := range undelegations {
		// Check if the application is already delegated to the gateway
		delegateeIdx := slices.Index(foundApp.DelegateeGatewayAddresses, undelegation.GatewayAddress)
		if delegateeIdx == -1 {
			appNotDelegatedFmt := "Application not delegated to gateway with address %q"
			logger.Info(fmt.Sprintf(appNotDelegatedFmt, undelegation.GatewayAddress))
			return types.ErrAppNotDelegated.Wrapf(appNotDelegatedFmt, undelegation.GatewayAddress)
		}

		// Remove the gateway from the application's delegatee gateway public keys
		foundApp.DelegateeGatewayAddresses = append(
			foundApp.DelegateeGatewayAddresses[:delegateeIdx],
			foundApp.DelegateeGatewayAddresses[delegateeIdx+1:]...,
		)

		k.deletePendingUndelegation(ctx, undelegation)

		logger.Info(fmt.Sprintf(
			"Successfully undelegated application from gateway for app: %+v",
			foundApp,
		))

		logger.Info(fmt.Sprintf("Emitting application redelegation event %v", undelegation))

		if err := ctx.EventManager().EmitTypedEvent(undelegation); err != nil {
			logger.Error(fmt.Sprintf("Failed to emit application redelegation event: %v", err))
			return err
		}
	}

	// Update the application store with the new delegation
	k.SetApplication(ctx, foundApp)

	return nil
}

func (k Keeper) deletePendingUndelegation(
	ctx context.Context,
	pendingUndelegation *types.Undelegation,
) {
	k.Logger().With("method", "deletePendingUndelegation").
		Info(fmt.Sprintf("Deleting pending undelegation %v", pendingUndelegation))

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(
		storeAdapter,
		types.KeyPrefix(types.PendingUndelegationsKeyPrefix),
	)

	hasPendingUndelegation := store.Has(types.PendingUndelegationsKey(pendingUndelegation))
	if hasPendingUndelegation {
		store.Delete(types.PendingUndelegationsKey(pendingUndelegation))
	}
}
