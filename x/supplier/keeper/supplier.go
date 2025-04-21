package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
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
}

// GetSupplier returns a supplier from its index
func (k Keeper) GetSupplier(
	ctx context.Context,
	supplierOperatorAddr string,
) (supplier sharedtypes.Supplier, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))

	supplierBz := store.Get(types.SupplierOperatorKey(supplierOperatorAddr))
	if supplierBz == nil {
		return supplier, false
	}

	k.cdc.MustUnmarshal(supplierBz, &supplier)

	initializeNilSupplierFields(&supplier)
	return supplier, true
}

// RemoveSupplier removes a supplier from the store
func (k Keeper) RemoveSupplier(ctx context.Context, supplierOperatorAddress string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
	store.Delete(types.SupplierOperatorKey(supplierOperatorAddress))
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

		initializeNilSupplierFields(&supplier)
		suppliers = append(suppliers, supplier)
	}

	return
}

// initializeNilSupplierFields initializes any nil fields in the supplier object
// to their default values.
// Adding `(gogoproto.nullable)=false` to the repeated proto fields
// acts on the underlying type and not the slice or map type. This means that the
// slice or map will be nil if no values are provided in the proto message.
// This function ensures that the supplier object has all fields initialized
// to their default values.
// TODO_TECHDEBT: This function is a workaround for the CosmosSDK codec treating
// empty slices and maps as nil. We should investigate how we can make the codec
// treat empty slices and maps as empty instead of nil.
// Refer to the following discussion for more context:
// https://github.com/pokt-network/poktroll/pull/1103#discussion_r1992258822
func initializeNilSupplierFields(supplier *sharedtypes.Supplier) {
	// The CosmosSDK codec treats empty slices and maps as nil, so we need to
	// ensure that they are initialized as empty.
	if supplier.Services == nil {
		supplier.Services = make([]*sharedtypes.SupplierServiceConfig, 0)
	}

	// Ensure that the supplier has at least one service config history entry.
	// This may be the case if the supplier was created at genesis.
	if supplier.ServiceConfigHistory == nil {
		supplier.ServiceConfigHistory = []*sharedtypes.ServiceConfigUpdate{
			{
				Services:             supplier.Services,
				EffectiveBlockHeight: 1,
			},
		}
	}
}

// GetAllSuppliersIterator returns a RecordIterator over all Supplier records.
func (k Keeper) GetAllSuppliersIterator(ctx context.Context) sharedtypes.RecordIterator[*sharedtypes.Supplier] {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
	supplierIterator := storetypes.KVStorePrefixIterator(store, []byte{})

	supplierUnmarshallerFn := getSupplierAccessorFn(k.cdc)
	return sharedtypes.NewRecordIterator(supplierIterator, supplierUnmarshallerFn)
}

// TODO_IMPROVE: Index suppliers by service ID
//func (k Keeper) GetAllSuppliersByServiceIDIterator(ctx, sdkContext, serviceId string) (suppliers []*sharedtypes.Supplier) {}

// getSupplierAccessorFn constructions a DataRecordAccessor function which:
// 1. Receives a serialized Supplier value bytes
// 2. Unmarshals it into a Supplier object
// 3. Initializes any nil fields in the Supplier object
// Returns:
// - A Supplier object and an error
func getSupplierAccessorFn(
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[*sharedtypes.Supplier] {
	return func(supplierBz []byte) (*sharedtypes.Supplier, error) {
		if supplierBz == nil {
			return nil, nil
		}

		var supplier sharedtypes.Supplier
		cdc.MustUnmarshal(supplierBz, &supplier)
		initializeNilSupplierFields(&supplier)
		return &supplier, nil
	}
}
