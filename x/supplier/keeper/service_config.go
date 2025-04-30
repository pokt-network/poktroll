package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// GetServiceConfigUpdatesIterator returns an iterator over service configuration
// updates with activation heights less than or equal to the provided current height.
//
// This function leverages the lexicographical ordering of big-endian encoded heights
// to efficiently filter configurations that should be active at the current height.
func (k Keeper) GetServiceConfigUpdatesIterator(
	ctx context.Context,
	serviceId string,
	currentHeight int64,
) sharedtypes.RecordIterator[*sharedtypes.ServiceConfigUpdate] {
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)

	startKeyBz := types.StringKey(serviceId)
	endKeyBz := types.StringKey(serviceId)

	// Append the currentHeight+1 in big endian format to create our upper bound.
	// Using currentHeight+1 makes the bound exclusive, so we get all heights <= currentHeight
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(currentHeight+1))
	endKeyBz = append(endKeyBz, heightBytes...)

	// Create an iterator for the range
	supplierServiceConfigIterator := serviceConfigUpdateStore.Iterator(startKeyBz, endKeyBz)

	serviceConfigUpdateAccessor := serviceConfigUpdateAccessorFn(k.cdc)
	return sharedtypes.NewRecordIterator(supplierServiceConfigIterator, serviceConfigUpdateAccessor)
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

	// This creates a closure that will be used by the RecordIterator to:
	// 1. Extract primary keys from the activation height index
	// 2. Use those keys to look up the full ServiceConfigUpdate objects in the primary store
	// 3. Return the unmarshaled ServiceConfigUpdate objects to the iterator consumer
	serviceConfigUpdateAccessor := serviceConfigUpdateFromPrimaryKeyAccessorFn(serviceConfigUpdateStore, k.cdc)
	return sharedtypes.NewRecordIterator(serviceConfigUpdateIterator, serviceConfigUpdateAccessor)
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

	serviceConfigAccessor := serviceConfigUpdateFromPrimaryKeyAccessorFn(serviceConfigUpdateStore, k.cdc)
	return sharedtypes.NewRecordIterator(serviceConfigUpdateIterator, serviceConfigAccessor)
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
	serviceConfigUpdateKey := types.ServiceConfigUpdateKey(serviceConfigUpdate)
	serviceConfigUpdateStore.Delete(serviceConfigUpdateKey)

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

// serviceConfigUpdateAccessorFn creates a function that unmarshals byte data
// into ServiceConfigUpdate objects.
//
// This creates a closure that can be used by the RecordIterator to convert raw
// bytes into typed ServiceConfigUpdate objects.
func serviceConfigUpdateAccessorFn(
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[*sharedtypes.ServiceConfigUpdate] {
	return func(serviceConfigUpdateBz []byte) (*sharedtypes.ServiceConfigUpdate, error) {
		var serviceConfigUpdate sharedtypes.ServiceConfigUpdate
		cdc.MustUnmarshal(serviceConfigUpdateBz, &serviceConfigUpdate)

		return &serviceConfigUpdate, nil
	}
}

// serviceConfigUpdateFromPrimaryKeyAccessorFn creates a function that retrieves a
// ServiceConfigUpdate from its primary key.
//
// This creates a closure that can be used by the RecordIterator to convert primary
// keys into the actual ServiceConfigUpdate objects they reference.
func serviceConfigUpdateFromPrimaryKeyAccessorFn(
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

// getServiceConfigUpdatesByServiceStore returns the KVStore that indexes service
// configuration updates by service ID.
// This enables efficiently finding all service configurations for a specific service.
func (k Keeper) getServiceConfigUpdatesByServiceStore(ctx context.Context, serviceId string) storetypes.KVStore {
	// Build the composite key for accessing service config updates for a specific service
	// Format: ServiceConfigUpdateKeyPrefix + serviceId
	key := make([]byte, 0)
	key = append(key, types.KeyPrefix(types.ServiceConfigUpdateKeyPrefix)...)
	key = append(key, types.StringKey(serviceId)...)

	// Prefix store for service config updates
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceStore := prefix.NewStore(storeAdapter, key)

	return serviceStore
}
