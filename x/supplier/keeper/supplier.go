package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SetAndIndexDehydratedSupplier stores a supplier record and indexes its relevant attributes for efficient querying.
// It modifies the Supplier structure onchain metadata to manage state bloat.
//
// The function:
// - Indexes service config updates for efficient retrieval
// - Indexes unstaking height (if applicable)
// - Indexes service usage metrics
// - Stores a dehydrated form of the supplier (without services and history)
func (k Keeper) SetAndIndexDehydratedSupplier(ctx context.Context, supplier sharedtypes.Supplier) {
	// Index service config updates for efficient retrieval
	k.indexSupplierServiceConfigUpdates(ctx, supplier)
	k.indexSupplierUnstakingHeight(ctx, supplier)
	k.indexSupplierServiceUsageMetrics(ctx, supplier)
	// Store the supplier in a dehydrated form to reduce state bloat
	k.SetDehydratedSupplier(ctx, supplier)
}

// SetDehydratedSupplier stores a dehydrated supplier in the store.
// It omits service details and history to reduce state bloat.
// This is useful when the service details are not needed for the current operation.
func (k Keeper) SetDehydratedSupplier(
	ctx context.Context,
	supplier sharedtypes.Supplier,
) {
	// Dehydrate the supplier to reduce state bloat.
	// These details can be hydrated just-in-time (when queried) using the indexes.
	supplier.Services = nil
	supplier.ServiceConfigHistory = nil
	supplier.ServiceUsageMetrics = nil
	k.storeSupplier(ctx, &supplier)
}

// GetDehydratedSupplier retrieves a dehydrated supplier.
// It omits hydrating a Supplier object with the available indexes (e.g. service config updates and unstaking height).
// Useful and more efficient when the service details aren't needed.
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
// Useful when all service details are needed.
func (k Keeper) GetSupplier(
	ctx context.Context,
	supplierOperatorAddr string,
) (supplier sharedtypes.Supplier, found bool) {
	// Retrieve a dehydrated supplier
	supplier, found = k.GetDehydratedSupplier(ctx, supplierOperatorAddr)
	if !found {
		return supplier, false
	}

	// Hydrate the supplier with service configurations
	k.hydrateSupplierServiceConfigs(ctx, &supplier)
	// Hydrate the supplier with service usage metrics
	k.hydrateSupplierServiceUsageMetrics(ctx, &supplier)

	return supplier, true
}

// RemoveSupplier deletes a supplier from the store and removes all associated indexes
func (k Keeper) RemoveSupplier(ctx context.Context, supplierOperatorAddress string) {
	// Remove all associated indexes
	k.removeSupplierServiceConfigUpdateIndexes(ctx, supplierOperatorAddress)
	k.removeSupplierUnstakingHeightIndex(ctx, supplierOperatorAddress)
	k.removeSupplierServiceUsageMetricsIndex(ctx, supplierOperatorAddress)

	// Delete the supplier from the store
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

// getSupplierServiceUsageMetricsIterator returns an iterator for supplier's service usage metrics
// - Creates an iterator that traverses all metrics for a specific supplier
// - Uses a prefix iterator to efficiently retrieve only metrics for the given supplier
// - Used when hydrating a supplier object with its complete service usage history
func (k Keeper) getSupplierServiceUsageMetricsIterator(
	ctx context.Context,
	supplierAddress string,
) sharedtypes.RecordIterator[sharedtypes.SupplierServiceUsageMetrics] {
	serviceUsageMetricsStore := k.getSupplierServiceUsageMetricsStore(ctx)
	supplierKey := types.StringKey(supplierAddress)

	serviceUsageMetricsIterator := storetypes.KVStorePrefixIterator(serviceUsageMetricsStore, supplierKey)

	serviceUsageMetricsAccessor := supplierUsageMetricsAccessorFn(k.cdc)
	return sharedtypes.NewRecordIterator(serviceUsageMetricsIterator, serviceUsageMetricsAccessor)
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

	supplier.ServiceConfigHistory = k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress, "")
	supplier.Services = supplier.GetActiveServiceConfigs(currentHeight)
}

// hydrateSupplierServiceUsageMetrics populates a supplier object with its service usage metrics
// - Retrieves metrics for all services the supplier provides or has provided
// - Loads metrics from the store and attaches them to the supplier object
// - Called during supplier retrieval to provide a complete supplier object with metrics
func (k Keeper) hydrateSupplierServiceUsageMetrics(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
) {
	// Hydrate the supplier's service usage metrics
	serviceUsageMetricsIterator := k.getSupplierServiceUsageMetricsIterator(ctx, supplier.OperatorAddress)
	for ; serviceUsageMetricsIterator.Valid(); serviceUsageMetricsIterator.Next() {
		supplierServiceUsageMetrics, err := serviceUsageMetricsIterator.Value()
		if err != nil {
			k.logger.Error(fmt.Sprintf("failed to get service usage metrics for supplier %s: %v", supplier.OperatorAddress, err))
			continue
		}

		supplier.ServiceUsageMetrics = append(
			supplier.ServiceUsageMetrics,
			supplierServiceUsageMetrics.ServiceUsageMetrics,
		)
	}
}

// GetSupplierActiveServiceConfig retrieves a supplier's active service configuration
// update for a specific service ID based on the current block height.

// The function:
// - Retrieves the supplier's service configuration history for the specified service ID
// - Returns the configuration that is active at the current block height
func (k Keeper) GetSupplierActiveServiceConfig(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	serviceId string,
) []*sharedtypes.SupplierServiceConfig {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Retrieve the supplier's service configuration history for the specified service ID.
	serviceConfigHistory := k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress, serviceId)
	// Determine which update is active at the current block height.
	return sharedtypes.GetActiveServiceConfigsFromHistory(serviceConfigHistory, currentHeight)
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

// storeSupplier marshals and stores the supplier record in the supplier store.
func (k Keeper) storeSupplier(ctx context.Context, supplier *sharedtypes.Supplier) {
	supplierBz := k.cdc.MustMarshal(supplier)
	supplierStore := k.getSupplierStore(ctx)
	supplierKey := types.SupplierOperatorKey(supplier.OperatorAddress)
	supplierStore.Set(supplierKey, supplierBz)
}

// GetServiceUsageMetrics retrieves usage metrics for a specific supplier and service
// - Returns metrics tracking relay count and compute units provided by a supplier
// - Returns initialized empty metrics if none exist for the supplier/service pair
func (k Keeper) GetServiceUsageMetrics(
	ctx context.Context,
	supplierAddressAddress,
	serviceId string,
) sharedtypes.ServiceUsageMetrics {
	serviceUsageMetricsStore := k.getSupplierServiceUsageMetricsStore(ctx)

	serviceUsageMetricsKey := types.ServiceUsageMetricsKey(supplierAddressAddress, serviceId)
	serviceUsageMetricsBz := serviceUsageMetricsStore.Get(serviceUsageMetricsKey)

	supplierServiceUsageMetrics := sharedtypes.SupplierServiceUsageMetrics{
		SupplierAddress: supplierAddressAddress,
		ServiceUsageMetrics: &sharedtypes.ServiceUsageMetrics{
			ServiceId: serviceId,
		},
	}

	if serviceUsageMetricsBz == nil {
		return *supplierServiceUsageMetrics.ServiceUsageMetrics
	}

	k.cdc.MustUnmarshal(serviceUsageMetricsBz, &supplierServiceUsageMetrics)
	return *supplierServiceUsageMetrics.ServiceUsageMetrics
}

// SetServiceUsageMetrics stores service usage metrics for a specific supplier and service
func (k Keeper) SetServiceUsageMetrics(
	ctx context.Context,
	supplierAddressAddress string,
	serviceUsageMetrics *sharedtypes.ServiceUsageMetrics,
) {
	serviceUsageMetricsStore := k.getSupplierServiceUsageMetricsStore(ctx)

	supplierServiceUsageMetrics := sharedtypes.SupplierServiceUsageMetrics{
		SupplierAddress:     supplierAddressAddress,
		ServiceUsageMetrics: serviceUsageMetrics,
	}

	serviceUsageMetricsBz := k.cdc.MustMarshal(&supplierServiceUsageMetrics)
	serviceUsageMetricsKey := types.ServiceUsageMetricsKey(supplierAddressAddress, serviceUsageMetrics.ServiceId)
	serviceUsageMetricsStore.Set(serviceUsageMetricsKey, serviceUsageMetricsBz)
}

// getSupplierServiceUsageMetricsStore returns the KVStore for supplier service usage metrics
func (k Keeper) getSupplierServiceUsageMetricsStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceUsageMetricsKeyPrefix))
}

// supplierUsageMetricsAccessorFn creates an accessor function for supplier service metrics
// - Used with RecordIterator to unmarshal binary metrics data
// - Validates that input bytes exist before attempting to unmarshal
// - Returns deserialized SupplierServiceUsageMetrics objects from storage
func supplierUsageMetricsAccessorFn(
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[sharedtypes.SupplierServiceUsageMetrics] {
	return func(serviceUsageMetricsBz []byte) (sharedtypes.SupplierServiceUsageMetrics, error) {
		if serviceUsageMetricsBz == nil {
			err := fmt.Errorf("expecting service usage metrics bytes to be non-nil")
			return sharedtypes.SupplierServiceUsageMetrics{}, err
		}

		var serviceUsageMetrics sharedtypes.SupplierServiceUsageMetrics
		cdc.MustUnmarshal(serviceUsageMetricsBz, &serviceUsageMetrics)
		return serviceUsageMetrics, nil
	}
}
