package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// GetServiceConfigUpdatesIterator returns an iterator over service configuration updates
// for a specific service ID.
//
// It provides access to all service configurations across all suppliers that are
// registered for the specified service, regardless of their activation status.
func (k Keeper) GetServiceConfigUpdatesIterator(
	ctx context.Context,
	serviceId string,
) sharedtypes.RecordIterator[*sharedtypes.ServiceConfigUpdate] {
	serviceConfigUpdatesStore := k.getServiceConfigUpdatesStore(ctx)

	serviceConfigUpdateIterator := storetypes.KVStorePrefixIterator(
		serviceConfigUpdatesStore,
		types.StringKey(serviceId),
	)

	serviceConfigUpdateAccessor := getServiceConfigUpdateFromBytesFn(k.cdc)
	return sharedtypes.NewRecordIterator(serviceConfigUpdateIterator, serviceConfigUpdateAccessor)
}

// GetActivatedServiceConfigUpdatesIterator returns an iterator over service configurations
// that become active at the specified block height.
//
// This is used primarily by the BeginBlocker to find all service configurations that
// need to be activated at the current block height.
func (k Keeper) GetActivatedServiceConfigUpdatesIterator(
	ctx context.Context,
	activationHeight int64,
) sharedtypes.RecordIterator[sharedtypes.ServiceConfigUpdate] {
	activationHeightStore := k.getServiceConfigUpdateActivationHeightStore(ctx)
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)

	serviceConfigUpdateIterator := storetypes.KVStorePrefixIterator(
		activationHeightStore,
		types.IntKey(activationHeight),
	)

	// TODO_IN_THIS_COMMIT: Add a comment explaining how getServiceConfigUpdateFromPrimaryKeyFn is used
	serviceConfigUpdateRetriever := getServiceConfigUpdateFromPrimaryKeyFn(serviceConfigUpdateStore, k.cdc)
	return sharedtypes.NewRecordIterator(serviceConfigUpdateIterator, serviceConfigUpdateRetriever)
}

// GetDeactivatedServiceConfigUpdatesIterator returns an iterator over service
// configuration updates that become inactive at the specified block height.
//
// This is used primarily by the EndBlocker to find all service configuration
// updates that need to be pruned from state at the current block height.
func (k Keeper) GetDeactivatedServiceConfigUpdatesIterator(
	ctx context.Context,
	deactivationHeight int64,
) sharedtypes.RecordIterator[sharedtypes.ServiceConfigUpdate] {
	deactivationHeightStore := k.getServiceConfigUpdateDeactivationHeightStore(ctx)
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)

	serviceConfigUpdateIterator := storetypes.KVStorePrefixIterator(
		deactivationHeightStore,
		types.IntKey(deactivationHeight),
	)

	serviceConfigRetriever := getServiceConfigUpdateFromPrimaryKeyFn(serviceConfigUpdateStore, k.cdc)
	return sharedtypes.NewRecordIterator(serviceConfigUpdateIterator, serviceConfigRetriever)
}

// deleteDeactivatedServiceConfigUpdate removes a deactivated service configuration
// update from all indexes.
//
// When a service configuration update is deactivated and has reached the pruning
// threshold, this function removes it from:
// 1. The primary service config store
// 2. The supplier-to-service index
// 3. The activation height index
// 4. The deactivation height index
func (k Keeper) deleteDeactivatedServiceConfigUpdate(
	ctx context.Context,
	serviceConfigUpdate sharedtypes.ServiceConfigUpdate,
) {
	// Delete from primary service config store
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)
	serviceConfigUpdatePrimaryKey := types.ServiceConfigUpdatePrimaryKey(serviceConfigUpdate)
	serviceConfigUpdateStore.Delete(serviceConfigUpdatePrimaryKey)

	// Delete from supplier-to-service index
	supplierServiceConfigUpdateStore := k.getSupplierServiceConfigUpdatesStore(ctx)
	supplierServiceConfigUpdateKey := types.SupplierServiceConfigUpdateKey(serviceConfigUpdate)
	supplierServiceConfigUpdateStore.Delete(supplierServiceConfigUpdateKey)

	// Delete from activation height index
	activationHeightStore := k.getServiceConfigUpdateActivationHeightStore(ctx)
	activationKey := types.ServiceConfigUpdateActivationHeightKey(serviceConfigUpdate)
	activationHeightStore.Delete(activationKey)

	// Delete from deactivation height index
	deactivationHeightStore := k.getServiceConfigUpdateDeactivationHeightStore(ctx)
	deactivationKey := types.ServiceConfigUpdateDeactivationHeightKey(serviceConfigUpdate)
	deactivationHeightStore.Delete(deactivationKey)
}

// getServiceConfigUpdateFromBytesFn creates a function that unmarshals byte data
// into ServiceConfigUpdate objects.
//
// This creates a closure that can be used by the RecordIterator to convert raw
// bytes into typed ServiceConfigUpdate objects.
func getServiceConfigUpdateFromBytesFn(
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[*sharedtypes.ServiceConfigUpdate] {
	return func(serviceConfigUpdateBz []byte) (*sharedtypes.ServiceConfigUpdate, error) {
		var serviceConfigUpdate sharedtypes.ServiceConfigUpdate
		cdc.MustUnmarshal(serviceConfigUpdateBz, &serviceConfigUpdate)

		return &serviceConfigUpdate, nil
	}
}

// getServiceConfigUpdateFromPrimaryKeyFn creates a function that retrieves a
// ServiceConfigUpdate from its primary key.
//
// This creates a closure that can be used by the RecordIterator to convert primary
// keys into the actual ServiceConfigUpdate objects they reference.
func getServiceConfigUpdateFromPrimaryKeyFn(
	serviceConfigPrimaryStore storetypes.KVStore,
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[sharedtypes.ServiceConfigUpdate] {
	return func(serviceConfigPrimaryKey []byte) (sharedtypes.ServiceConfigUpdate, error) {
		serviceConfigBz := serviceConfigPrimaryStore.Get(serviceConfigPrimaryKey)
		if serviceConfigBz == nil {
			return sharedtypes.ServiceConfigUpdate{}, fmt.Errorf("expected service config update to exist for key: %v", serviceConfigPrimaryKey)
		}

		var serviceConfigUpdate sharedtypes.ServiceConfigUpdate
		cdc.MustUnmarshal(serviceConfigBz, &serviceConfigUpdate)

		return serviceConfigUpdate, nil
	}
}

// getServiceConfigUpdatesStore returns the KVStore for the primary service
// configuration update records.
func (k Keeper) getServiceConfigUpdatesStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceConfigUpdateKeyPrefix))
}

// getServiceConfigUpdateActivationHeightStore returns the KVStore that indexes
// service configuration updates by activation height.
// This enables efficiently finding all configurations that become active at a specific
// height without needing to iterate over and unmarshal all the corresponding suppliers.
func (k Keeper) getServiceConfigUpdateActivationHeightStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceConfigUpdateActivationHeightKeyPrefix))
}

// getServiceConfigUpdateDeactivationHeightStore returns the KVStore that indexes
// service configuration updates by deactivation height.
// This enables efficiently finding all configurations that become inactive at a specific
// height without needing to iterate over and unmarshal all the corresponding suppliers.
func (k Keeper) getServiceConfigUpdateDeactivationHeightStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ServiceConfigUpdateDeactivationHeightKeyPrefix))
}

// getSupplierServiceConfigUpdatesStore returns the KVStore that indexes service
// configuration updates by supplier operator address.
// This enables efficiently finding all service configurations for a specific supplier.
func (k Keeper) getSupplierServiceConfigUpdatesStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierServiceConfigUpdateKeyPrefix))
}
