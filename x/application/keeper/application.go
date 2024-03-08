package keeper

import (
	"context"
	"fmt"

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
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx context.Context,
	appAddr string,
) (app types.Application, found bool) {
	logger := k.Logger().With("Func", "GetApplication").With("appAddr", appAddr)

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))

	appBz := store.Get(types.ApplicationKey(appAddr))
	if appBz == nil {
		return app, false
	}

	logger.Info(fmt.Sprintf("found application with address: %s", appAddr))

	k.cdc.MustUnmarshal(appBz, &app)
	return app, true
}

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(ctx context.Context, appAddr string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(appAddr))
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
		apps = append(apps, app)
	}

	return
}
