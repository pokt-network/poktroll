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
	k.cache.Services[service.Id] = &service
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	serviceBz := k.cdc.MustMarshal(&service)
	store.Set(types.ServiceKey(service.Id), serviceBz)
}

// GetService returns a service from its index
func (k Keeper) GetService(
	ctx context.Context,
	serviceId string,
) (service sharedtypes.Service, found bool) {
	if service, found := k.cache.Services[service.Id]; found {
		k.logger.Info("-----Service cache hit-----")
		return *service, true
	}
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))

	serviceBz := store.Get(types.ServiceKey(serviceId))
	if serviceBz == nil {
		return service, false
	}

	k.cdc.MustUnmarshal(serviceBz, &service)
	k.cache.Services[service.Id] = &service
	return service, true
}

// RemoveService removes a service from the store
func (k Keeper) RemoveService(
	ctx context.Context,
	serviceId string,
) {
	delete(k.cache.Services, serviceId)
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	store.Delete(types.ServiceKey(serviceId))
}

// GetAllServices returns all services
func (k Keeper) GetAllServices(ctx context.Context) (services []sharedtypes.Service) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var service sharedtypes.Service
		k.cdc.MustUnmarshal(iterator.Value(), &service)
		k.cache.Services[service.Id] = &service
		services = append(services, service)
	}

	return
}
