package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/application/types"
)

// SetApplication set a specific application in the store from its index
func (k Keeper) SetApplication(ctx context.Context, application types.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	appBz := k.cdc.MustMarshal(&application)
	store.Set(types.ApplicationKey(application.Address), appBz)
	k.applicationsCache.Set(application.Address, application)
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx context.Context,
	appAddr string,
) (app types.Application, found bool) {
	if app, found := k.applicationsCache.Get(appAddr); found {
		k.logger.Info("-----Application cache hit-----")
		return app, true
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))

	appBz := store.Get(types.ApplicationKey(appAddr))
	if appBz == nil {
		return app, false
	}

	k.cdc.MustUnmarshal(appBz, &app)

	// Ensure that the PendingUndelegations is an empty map and not nil when
	// unmarshalling an app that has no pending undelegations.
	if app.PendingUndelegations == nil {
		app.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
	}

	// Ensure that the DelegateeGatewayAddresses is an empty slice and not nil
	// when unmarshalling an app that has no delegations.
	if app.DelegateeGatewayAddresses == nil {
		app.DelegateeGatewayAddresses = make([]string, 0)
	}

	k.applicationsCache.Set(appAddr, app)

	return app, true
}

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(ctx context.Context, appAddr string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(appAddr))
	k.applicationsCache.Delete(appAddr)
}

// GetAllApplications returns all application
func (k Keeper) GetAllApplications(ctx context.Context) (apps []types.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var app types.Application
		k.cdc.MustUnmarshal(iterator.Value(), &app)

		// Ensure that the PendingUndelegations is an empty map and not nil when
		// unmarshalling an app that has no pending undelegations.
		if app.PendingUndelegations == nil {
			app.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
		}

		k.applicationsCache.Set(app.Address, app)

		apps = append(apps, app)
	}

	return
}
