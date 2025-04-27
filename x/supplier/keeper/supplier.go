package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SetSupplier stores a supplier state and indexes its relevant attributes for efficient querying.
// It processes the supplier's service configurations for indexing.
//
// The function:
// - Indexes service config updates for efficient retrieval
// - Indexes unstaking height (if applicable)
// - Stores a dehydrated form of the supplier (without services and history)
func (k Keeper) SetSupplier(ctx context.Context, supplier sharedtypes.Supplier) {
	k.indexSupplierServiceConfigUpdates(ctx, supplier)
	k.indexSupplierUnstakingHeight(ctx, supplier)

	// Store the supplier without service details to reduce state bloat
	// These details will be hydrated on-demand via the service config indexes
	supplier.Services = nil
	supplier.ServiceConfigHistory = nil
	supplierBz := k.cdc.MustMarshal(&supplier)

	supplierStore := k.getSupplierStore(ctx)
	supplierKey := types.SupplierOperatorKey(supplier.OperatorAddress)
	supplierStore.Set(supplierKey, supplierBz)
}

// GetDehydratedSupplier retrieves a supplier from the store without its service
// configurations and service config history.
// This is more efficient when the service details aren't needed.
func (k Keeper) GetDehydratedSupplier(
	ctx context.Context,
	supplierOperatorAddr string,
) (supplier sharedtypes.Supplier, found bool) {
	supplierStore := k.getSupplierStore(ctx)

	supplierKey := types.SupplierOperatorKey(supplierOperatorAddr)
	supplierBz := supplierStore.Get(supplierKey)
	if supplierBz == nil {
		return supplier, false
	}

	k.cdc.MustUnmarshal(supplierBz, &supplier)

	return supplier, true
}

// GetSupplier retrieves a fully hydrated supplier from the store, including
// its service configurations and service config updates history.
func (k Keeper) GetSupplier(
	ctx context.Context,
	supplierOperatorAddr string,
) (supplier sharedtypes.Supplier, found bool) {
	supplier, found = k.GetDehydratedSupplier(ctx, supplierOperatorAddr)
	if !found {
		return supplier, false
	}

	k.hydrateSupplierServiceConfigs(ctx, &supplier)

	return supplier, true
}

// RemoveSupplier deletes a supplier from the store and removes all associated indexes
func (k Keeper) RemoveSupplier(ctx context.Context, supplierOperatorAddress string) {
	k.removeSupplierServiceConfigUpdateIndexes(ctx, supplierOperatorAddress)
	k.removeSupplierUnstakingHeightIndexes(ctx, supplierOperatorAddress)

	supplierStore := k.getSupplierStore(ctx)
	supplierKey := types.SupplierOperatorKey(supplierOperatorAddress)
	supplierStore.Delete(supplierKey)
}

// GetAllSuppliers returns all suppliers stored in the blockchain state
// Each supplier is fully hydrated with its service configurations and history.
func (k Keeper) GetAllSuppliers(ctx context.Context) (suppliers []sharedtypes.Supplier) {
	supplierStore := k.getSupplierStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(supplierStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var supplier sharedtypes.Supplier
		k.cdc.MustUnmarshal(iterator.Value(), &supplier)
		k.hydrateSupplierServiceConfigs(ctx, &supplier)

		suppliers = append(suppliers, supplier)
	}

	return suppliers
}

// GetAllUnstakingSuppliersIterator returns an iterator for all suppliers that are
// currently unstaking.
// It is used to process suppliers that have completed their unbonding period.
func (k Keeper) GetAllUnstakingSuppliersIterator(
	ctx context.Context,
) storetypes.Iterator {
	supplierUnstakingHeightStore := k.getSupplierUnstakingHeightStore(ctx)

	return storetypes.KVStorePrefixIterator(supplierUnstakingHeightStore, []byte{})
}

// hydrateSupplierServiceConfigs populates a supplier with its service configurations
// based on the current block height.
//
// The function:
// - Retrieves the supplier's service configuration history
// - Determines which configurations are active at the current block height
// - Sets the supplier's active services
func (k Keeper) hydrateSupplierServiceConfigs(ctx context.Context, supplier *sharedtypes.Supplier) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	supplier.ServiceConfigHistory = k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress)
	supplier.Services = supplier.GetActiveServiceConfigs(currentHeight)
}

// getSupplierStore returns a KVStore for the supplier data
func (k Keeper) getSupplierStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierOperatorKeyPrefix))
}

// getSupplierUnstakingHeightStore returns a KVStore for the supplier unstaking height index
func (k Keeper) getSupplierUnstakingHeightStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierUnstakingHeightKeyPrefix))
}

// GetAllDeprecatedSuppliers returns all suppliers in their deprecated form
// (i.e. Prior to v0.1.8)
// TODO_FOLLOWUP: Remove this function after v0.1.8 upgrade
func (k Keeper) GetAllDeprecatedSuppliers(ctx context.Context) (suppliers []sharedtypes.SupplierDeprecated) {
	supplierStore := k.getSupplierStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(supplierStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var supplier sharedtypes.SupplierDeprecated
		k.cdc.MustUnmarshal(iterator.Value(), &supplier)

		suppliers = append(suppliers, supplier)
	}

	return suppliers
}
