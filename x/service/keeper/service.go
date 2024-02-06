package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SetService set a specific service in the store from its index
func (k Keeper) SetService(ctx context.Context, service sharedtypes.Service) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	b := k.cdc.MustMarshal(&service)
	store.Set(types.ServiceKey(
		service.Id,
	), b)
}

// GetService returns a service from its index
func (k Keeper) GetService(
	ctx context.Context,
	index string,

) (val sharedtypes.Service, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))

	b := store.Get(types.ServiceKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveService removes a service from the store
func (k Keeper) RemoveService(
	ctx context.Context,
	index string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	store.Delete(types.ServiceKey(
		index,
	))
}

// GetAllService returns all service
func (k Keeper) GetAllService(ctx context.Context) (list []sharedtypes.Service) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val sharedtypes.Service
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
