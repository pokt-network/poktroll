package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SetSupplier set a specific supplier in the store from its index
func (k Keeper) SetSupplier(ctx context.Context, supplier shared.Supplier) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyPrefix))
	supplierBz := k.cdc.MustMarshal(&supplier)
	store.Set(types.SupplierKey(
		supplier.Address,
	), supplierBz)
}

// GetSupplier returns a supplier from its index
func (k Keeper) GetSupplier(
	ctx context.Context,
	supplierAddr string,
) (supplier shared.Supplier, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyPrefix))

	supplierBz := store.Get(types.SupplierKey(supplierAddr))
	if supplierBz == nil {
		return supplier, false
	}

	k.cdc.MustUnmarshal(supplierBz, &supplier)
	return supplier, true
}

// RemoveSupplier removes a supplier from the store
func (k Keeper) RemoveSupplier(ctx context.Context, address string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyPrefix))
	store.Delete(types.SupplierKey(address))
}

// GetAllSuppliers returns all supplier
func (k Keeper) GetAllSuppliers(ctx context.Context) (suppliers []shared.Supplier) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var supplier shared.Supplier
		k.cdc.MustUnmarshal(iterator.Value(), &supplier)
		suppliers = append(suppliers, supplier)
	}

	return
}

// TODO_MAINNET: Index suppliers by service so we can easily query k.GetAllSuppliers(ctx, Service)
// func (k Keeper) GetAllSuppliers(ctx, sdkContext, serviceId string) (suppliers []sharedtypes.Supplier) {}
