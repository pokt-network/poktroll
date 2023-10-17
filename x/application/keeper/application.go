package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"pocket/x/application/types"
)

// SetApplication set a specific application in the store from its index
func (k Keeper) SetApplication(ctx sdk.Context, application types.Application) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))
	b := k.cdc.MustMarshal(&application)
	store.Set(types.ApplicationKey(
		application.Address,
	), b)
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx sdk.Context,
	address string,

) (val types.Application, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))

	b := store.Get(types.ApplicationKey(
		address,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(
	ctx sdk.Context,
	address string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(
		address,
	))
}

// GetAllApplication returns all application
func (k Keeper) GetAllApplication(ctx sdk.Context) (list []types.Application) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Application
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
