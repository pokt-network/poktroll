package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "pocket/x/shared/types"
	"pocket/x/supplier/types"
)

// SetSupplier set a specific supplier in the store from its index
func (k Keeper) SetSupplier(ctx sdk.Context, supplier sharedtypes.Supplier) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SupplierKeyPrefix))
	b := k.cdc.MustMarshal(&supplier)
	store.Set(types.SupplierKey(
		supplier.Address,
	), b)
}

// GetSupplier returns a supplier from its index
func (k Keeper) GetSupplier(
	ctx sdk.Context,
	supplierAddr string,

) (supplier sharedtypes.Supplier, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SupplierKeyPrefix))

	b := store.Get(types.SupplierKey(
		supplierAddr,
	))
	if b == nil {
		return supplier, false
	}

	k.cdc.MustUnmarshal(b, &supplier)
	return supplier, true
}

// RemoveSupplier removes a supplier from the store
func (k Keeper) RemoveSupplier(
	ctx sdk.Context,
	supplierAddr string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SupplierKeyPrefix))
	store.Delete(types.SupplierKey(
		supplierAddr,
	))
}

// GetAllSupplier returns all supplier
func (k Keeper) GetAllSupplier(ctx sdk.Context) (suppliers []sharedtypes.Supplier) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SupplierKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var supplier sharedtypes.Supplier
		k.cdc.MustUnmarshal(iterator.Value(), &supplier)
		suppliers = append(suppliers, supplier)
	}

	return
}

// TODO_OPTIMIZE: Index suppliers by serviceId so we can easily query `k.GetAllSupplier(ctx, ServiceId)`
// func (k Keeper) GetAllSupplier(ctx, sdkContext, serviceId string) (suppliers []sharedtypes.Supplier) {}
