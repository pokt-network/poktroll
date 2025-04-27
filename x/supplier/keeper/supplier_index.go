package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// indexSupplierServiceConfigUpdates maintains multiple indices for efficient
// lookups of service configuration updates.
//
// This function indexes the supplier's service configurations in four different ways:
// 1. By primary key for direct access to service config updates
// 2. By supplier operator address for finding all services a supplier provides
// 3. By activation height for efficiently finding all services becoming active at a specific height
// 4. By deactivation height (if specified) for efficiently finding all services becoming inactive
//
// Each index stores a reference to the primary key, which allows efficient retrieval
// of the full configuration data when needed.
func (k Keeper) indexSupplierServiceConfigUpdates(
	ctx context.Context,
	supplier sharedtypes.Supplier,
) {
	// Get all the necessary stores
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)
	supplierServiceConfigUpdateStore := k.getSupplierServiceConfigUpdatesStore(ctx)
	serviceConfigUpdateActivationHeightStore := k.getServiceConfigUpdateActivationHeightStore(ctx)
	serviceConfigUpdateDeactivationHeightStore := k.getServiceConfigUpdateDeactivationHeightStore(ctx)

	// Index each service config update in the supplier's history
	for _, serviceConfigUpdate := range supplier.ServiceConfigHistory {
		// Serialize the config update
		serviceConfigBz := k.cdc.MustMarshal(serviceConfigUpdate)

		// Index 1: Primary key -> service config data
		serviceConfigPrimaryKey := types.ServiceConfigUpdateKey(*serviceConfigUpdate)
		serviceConfigUpdateStore.Set(serviceConfigPrimaryKey, serviceConfigBz)

		// Index 2: Supplier operator address -> primary key
		supplierServiceConfigKey := types.SupplierServiceConfigUpdateKey(*serviceConfigUpdate)
		supplierServiceConfigUpdateStore.Set(supplierServiceConfigKey, serviceConfigPrimaryKey)

		// Index 3: Activation height -> primary key
		serviceConfigActivationKey := types.ServiceConfigUpdateActivationHeightKey(*serviceConfigUpdate)
		serviceConfigUpdateActivationHeightStore.Set(serviceConfigActivationKey, serviceConfigPrimaryKey)

		// Index 4: Deactivation height -> primary key (only if deactivation is scheduled)
		if serviceConfigUpdate.DeactivationHeight != 0 {
			serviceConfigDeactivationKey := types.ServiceConfigUpdateDeactivationHeightKey(*serviceConfigUpdate)
			serviceConfigUpdateDeactivationHeightStore.Set(serviceConfigDeactivationKey, serviceConfigPrimaryKey)
		}
	}
}

// indexSupplierUnstakingHeight maintains an index of suppliers that are currently
// in the unbonding period.
//
// This function either adds or removes a supplier from the unstaking height index
// depending on whether the supplier is currently unbonding:
// - If the supplier is unbonding (UnstakeSessionEndHeight > 0), it's added to the index
// - If the supplier is not unbonding, it's removed from the index
//
// This index enables the EndBlocker to efficiently find suppliers whose unbonding period
// has completed without iterating over and unmarshaling all suppliers in the store.
func (k Keeper) indexSupplierUnstakingHeight(
	ctx context.Context,
	supplier sharedtypes.Supplier,
) {
	supplierUnstakingHeightStore := k.getSupplierUnstakingHeightStore(ctx)

	supplierOperatorKey := types.SupplierOperatorKey(supplier.OperatorAddress)
	if supplier.IsUnbonding() {
		// Add to unstaking index if supplier is unbonding
		supplierUnstakingHeightStore.Set(supplierOperatorKey, []byte(supplier.OperatorAddress))
	} else {
		// Remove from unstaking index if supplier is not unbonding
		supplierUnstakingHeightStore.Delete(supplierOperatorKey)
	}
}

// getSupplierServiceConfigUpdates retrieves all service configuration updates for a specific supplier.
//
// This function uses the supplier-to-service index to efficiently find all service
// configurations associated with the given supplier operator address, without needing
// to scan and unmarshal the entire service configuration updates store.
func (k Keeper) getSupplierServiceConfigUpdates(
	ctx context.Context,
	supplierOperatorAddress string,
) []*sharedtypes.ServiceConfigUpdate {
	// Get the necessary stores
	supplierServiceConfigUpdateStore := k.getSupplierServiceConfigUpdatesStore(ctx)
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)

	// Create iterator for the supplier's service configs
	supplierServiceConfigIterator := storetypes.KVStorePrefixIterator(
		supplierServiceConfigUpdateStore,
		types.StringKey(supplierOperatorAddress),
	)
	defer supplierServiceConfigIterator.Close()

	// Collect all service configuration updates
	serviceConfigUpdates := make([]*sharedtypes.ServiceConfigUpdate, 0)
	for ; supplierServiceConfigIterator.Valid(); supplierServiceConfigIterator.Next() {
		// Get the primary key from the supplier index
		serviceConfigPrimaryKey := supplierServiceConfigIterator.Value()
		// Use the primary key to get the actual service config data
		serviceConfigBz := serviceConfigUpdateStore.Get(serviceConfigPrimaryKey)

		// Unmarshal and collect the service config
		var serviceConfig sharedtypes.ServiceConfigUpdate
		k.cdc.MustUnmarshal(serviceConfigBz, &serviceConfig)
		serviceConfigUpdates = append(serviceConfigUpdates, &serviceConfig)
	}

	return serviceConfigUpdates
}

// removeSupplierServiceConfigUpdateIndexes removes all service configuration indexes for a supplier.
//
// This function is called when a supplier is completely removed from the state,
// typically after the unbonding period has completed. It removes:
// 1. All entries from the activation height index for this supplier's services
// 2. All entries from the deactivation height index for this supplier's services
// 3. All primary data entries for this supplier's services
// 4. All entries from the supplier-to-service index for this supplier
func (k Keeper) removeSupplierServiceConfigUpdateIndexes(
	ctx context.Context,
	supplierOperatorAddress string,
) {
	// Get all the necessary stores
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)
	supplierServiceConfigUpdateStore := k.getSupplierServiceConfigUpdatesStore(ctx)
	serviceConfigUpdateActivationHeightStore := k.getServiceConfigUpdateActivationHeightStore(ctx)
	serviceConfigUpdateDeactivationHeightStore := k.getServiceConfigUpdateDeactivationHeightStore(ctx)

	// Create iterator for the supplier's service configs
	supplierServiceConfigsIterator := storetypes.KVStorePrefixIterator(
		supplierServiceConfigUpdateStore,
		types.StringKey(supplierOperatorAddress),
	)

	// Track all keys that need to be deleted from the supplier index
	supplierServiceConfigKeys := make([][]byte, 0)

	// First pass: remove entries from activation/deactivation indices and primary store
	for ; supplierServiceConfigsIterator.Valid(); supplierServiceConfigsIterator.Next() {
		// Store the key for later deletion from the supplier index
		supplierServiceConfigKeys = append(supplierServiceConfigKeys, supplierServiceConfigsIterator.Key())

		// Get the service config using its primary key
		serviceConfigPrimaryKey := supplierServiceConfigsIterator.Value()
		serviceConfigBz := serviceConfigUpdateStore.Get(serviceConfigPrimaryKey)
		var serviceConfigUpdate sharedtypes.ServiceConfigUpdate
		k.cdc.MustUnmarshal(serviceConfigBz, &serviceConfigUpdate)

		// Delete from activation height index
		serviceConfigActivationKey := types.ServiceConfigUpdateActivationHeightKey(serviceConfigUpdate)
		serviceConfigUpdateActivationHeightStore.Delete(serviceConfigActivationKey)

		// Delete from deactivation height index
		serviceConfigDeactivationKey := types.ServiceConfigUpdateDeactivationHeightKey(serviceConfigUpdate)
		serviceConfigUpdateDeactivationHeightStore.Delete(serviceConfigDeactivationKey)

		// Delete from primary store
		serviceConfigUpdateStore.Delete(serviceConfigPrimaryKey)
	}
	supplierServiceConfigsIterator.Close()

	// Second pass: remove entries from the supplier-to-service index
	for _, supplierServiceConfigKey := range supplierServiceConfigKeys {
		supplierServiceConfigUpdateStore.Delete(supplierServiceConfigKey)
	}
}

// removeSupplierUnstakingHeightIndexes removes a supplier from the unstaking height index.
//
// This function is called when a supplier is completely removed from the state or
// when they re-stake, cancelling their unbonding period.
func (k Keeper) removeSupplierUnstakingHeightIndexes(
	ctx context.Context,
	supplierOperatorAddress string,
) {
	supplierUnstakingHeightStore := k.getSupplierUnstakingHeightStore(ctx)

	supplierUnstakeKey := types.SupplierOperatorKey(supplierOperatorAddress)
	supplierUnstakingHeightStore.Delete(supplierUnstakeKey)
}
