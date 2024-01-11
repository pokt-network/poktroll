package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SetService set a specific service in the store from its index
func (k Keeper) SetService(ctx sdk.Context, service sharedtypes.Service) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ServiceKeyPrefix))
	b := k.cdc.MustMarshal(&service)
	store.Set(types.ServiceKey(
		service.Id,
	), b)
}

// GetService returns a service from its index
func (k Keeper) GetService(
	ctx sdk.Context,
	id string,
) (val sharedtypes.Service, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ServiceKeyPrefix))

	b := store.Get(types.ServiceKey(
		id,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveService removes a service from the store
func (k Keeper) RemoveService(
	ctx sdk.Context,
	address string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ServiceKeyPrefix))
	store.Delete(types.ServiceKey(
		address,
	))
}

// GetAllServices returns all services
func (k Keeper) GetAllServices(ctx sdk.Context) (list []sharedtypes.Service) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ServiceKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		val := sharedtypes.Service{}
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
