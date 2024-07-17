package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/x/application/types"
)

// SetApplication set a specific application in the store from its index
func (k Keeper) SetApplication(ctx context.Context, application application.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	appBz := k.cdc.MustMarshal(&application)
	store.Set(types.ApplicationKey(application.Address), appBz)
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx context.Context,
	appAddr string,
) (app application.Application, found bool) {
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
		app.PendingUndelegations = make(map[uint64]application.UndelegatingGatewayList)
	}

	return app, true
}

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(ctx context.Context, appAddr string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(appAddr))
}

// GetAllApplications returns all application
func (k Keeper) GetAllApplications(ctx context.Context) (apps []application.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var app application.Application
		k.cdc.MustUnmarshal(iterator.Value(), &app)

		// Ensure that the PendingUndelegations is an empty map and not nil when
		// unmarshalling an app that has no pending undelegations.
		if app.PendingUndelegations == nil {
			app.PendingUndelegations = make(map[uint64]application.UndelegatingGatewayList)
		}

		apps = append(apps, app)
	}

	return
}
