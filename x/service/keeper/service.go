package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/x/service/types"
)

// SetService set a specific service in the store from its index
func (k Keeper) SetService(ctx context.Context, service shared.Service) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	serviceBz := k.cdc.MustMarshal(&service)
	store.Set(types.ServiceKey(service.Id), serviceBz)
}

// GetService returns a service from its index
func (k Keeper) GetService(
	ctx context.Context,
	serviceId string,
) (service shared.Service, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))

	serviceBz := store.Get(types.ServiceKey(serviceId))
	if serviceBz == nil {
		return service, false
	}

	k.cdc.MustUnmarshal(serviceBz, &service)
	return service, true
}

// RemoveService removes a service from the store
func (k Keeper) RemoveService(
	ctx context.Context,
	serviceId string,
) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	store.Delete(types.ServiceKey(serviceId))
}

// GetAllServices returns all services
func (k Keeper) GetAllServices(ctx context.Context) (services []shared.Service) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var service shared.Service
		k.cdc.MustUnmarshal(iterator.Value(), &service)
		services = append(services, service)
	}

	return
}
