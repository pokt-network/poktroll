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
	b := k.cdc.MustMarshal(&application)
	store.Set(types.ApplicationKey(
		application.Address,
	), b)
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx context.Context,
	appAddr string,

) (app types.Application, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))

	b := store.Get(types.ApplicationKey(
		appAddr,
	))
	if b == nil {
		return app, false
	}

	k.cdc.MustUnmarshal(b, &app)
	return app, true
}

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(
	ctx context.Context,
	appAddr string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(
		appAddr,
	))
}

// GetAllApplication returns all application
func (k Keeper) GetAllApplication(ctx context.Context) (apps []types.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Application
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		apps = append(apps, val)
	}

	return
}
