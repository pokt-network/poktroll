package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SetService set a specific service in the store from its index.
func (k Keeper) SetService(ctx sdk.Context, service sharedtypes.Service) {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.ServiceKeyPrefix),
	)
	serviceBz := k.cdc.MustMarshal(&service)
	store.Set(types.ServiceKey(
		service.Id,
	), serviceBz)
}

// GetService returns a service from the store by its index.
func (k Keeper) GetService(
	ctx sdk.Context,
	serviceID string,
) (service sharedtypes.Service, found bool) {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.ServiceKeyPrefix),
	)

	serviceBz := store.Get(types.ServiceKey(
		serviceID,
	))
	if serviceBz == nil {
		return service, false
	}

	k.cdc.MustUnmarshal(serviceBz, &service)
	return service, true
}

// RemoveService removes a service from the store.
func (k Keeper) RemoveService(
	ctx sdk.Context,
	serviceID string,
) {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.ServiceKeyPrefix),
	)
	store.Delete(types.ServiceKey(
		serviceID,
	))
}

// GetAllServices returns all services from the store.
func (k Keeper) GetAllServices(ctx sdk.Context) (list []sharedtypes.Service) {
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		types.KeyPrefix(types.ServiceKeyPrefix),
	)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		service := sharedtypes.Service{}
		k.cdc.MustUnmarshal(iterator.Value(), &service)
		list = append(list, service)
	}

	return
}
