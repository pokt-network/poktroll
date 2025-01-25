package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SetSupplier set a specific supplier in the store from its index
func (k Keeper) SetSupplier(ctx context.Context, supplier sharedtypes.Supplier) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
	supplierBz := k.cdc.MustMarshal(&supplier)
	store.Set(types.SupplierOperatorKey(
		supplier.OperatorAddress,
	), supplierBz)
	k.suppliersCache.Set(supplier.OperatorAddress, supplier)
}

// GetSupplier returns a supplier from its index
func (k Keeper) GetSupplier(
	ctx context.Context,
	supplierOperatorAddr string,
) (supplier sharedtypes.Supplier, found bool) {
	if supplier, found := k.suppliersCache.Get(supplierOperatorAddr); found {
		k.logger.Info("-----Supplier cache hit-----")
		return supplier, true
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))

	supplierBz := store.Get(types.SupplierOperatorKey(supplierOperatorAddr))
	if supplierBz == nil {
		return supplier, false
	}

	k.cdc.MustUnmarshal(supplierBz, &supplier)
	k.suppliersCache.Set(supplier.OperatorAddress, supplier)
	return supplier, true
}

// RemoveSupplier removes a supplier from the store
func (k Keeper) RemoveSupplier(ctx context.Context, supplierOperatorAddress string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
	store.Delete(types.SupplierOperatorKey(supplierOperatorAddress))
	k.suppliersCache.Delete(supplierOperatorAddress)
}

// GetAllSuppliers returns all supplier
func (k Keeper) GetAllSuppliers(ctx context.Context) (suppliers []sharedtypes.Supplier) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var supplier sharedtypes.Supplier
		k.cdc.MustUnmarshal(iterator.Value(), &supplier)
		k.suppliersCache.Set(supplier.OperatorAddress, supplier)
		suppliers = append(suppliers, supplier)
	}

	return
}

func (k Keeper) ClearCache() {
	k.suppliersCache.Clear()
}

// TODO_OPTIMIZE: Index suppliers by service ID
// func (k Keeper) GetAllSuppliers(ctx, sdkContext, serviceId string) (suppliers []sharedtypes.Supplier) {}
