package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SetApplication set a specific application in the store from its index
// and updates all related application indexes.
func (k Keeper) SetApplication(ctx context.Context, application types.Application) {
	k.indexApplicationUnstaking(ctx, application)
	k.indexApplicationTransfer(ctx, application)
	k.indexApplicationDelegations(ctx, application)
	k.indexApplicationUndelegations(ctx, application)

	applicationStore := k.getApplicationStore(ctx)
	appBz := k.cdc.MustMarshal(&application)
	applicationStore.Set(types.ApplicationKey(application.Address), appBz)
}

// GetApplication returns a application from its index
// It initializes any nil fields to empty collections when found
func (k Keeper) GetApplication(
	ctx context.Context,
	appAddr string,
) (app types.Application, found bool) {
	applicationStore := k.getApplicationStore(ctx)

	appBz := applicationStore.Get(types.ApplicationKey(appAddr))
	if appBz == nil {
		return app, false
	}

	k.cdc.MustUnmarshal(appBz, &app)

	// Ensure that the PendingUndelegations is an empty map and not nil when
	// unmarshalling an app that has no pending undelegations.
	if app.PendingUndelegations == nil {
		app.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
	}

	// Ensure that the DelegateeGatewayAddresses is an empty slice and not nil
	// when unmarshalling an app that has no delegations.
	if app.DelegateeGatewayAddresses == nil {
		app.DelegateeGatewayAddresses = make([]string, 0)
	}

	return app, true
}

// RemoveApplication removes an application from the store and all related application indexes.
func (k Keeper) RemoveApplication(ctx context.Context, application types.Application) {
	k.removeApplicationUnstakingIndex(ctx, application.Address)
	k.removeApplicationTransferIndex(ctx, application.Address)
	k.removeApplicationUndelegationIndexes(ctx, application.Address)

	for _, gatewayAddress := range application.DelegateeGatewayAddresses {
		k.removeApplicationDelegationIndex(ctx, application.Address, gatewayAddress)
	}

	applicationStore := k.getApplicationStore(ctx)
	applicationStore.Delete(types.ApplicationKey(application.Address))
}

// GetAllApplications returns all applications
func (k Keeper) GetAllApplications(ctx context.Context) (apps []types.Application) {
	applicationStore := k.getApplicationStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(applicationStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var app types.Application
		k.cdc.MustUnmarshal(iterator.Value(), &app)

		// Ensure that the PendingUndelegations is an empty map and not nil when
		// unmarshalling an app that has no pending undelegations.
		if app.PendingUndelegations == nil {
			app.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
		}

		apps = append(apps, app)
	}

	return
}

// GetAllUnstakingApplicationsIterator returns an iterator for all applications
// that are currently unstaking.
func (k Keeper) GetAllUnstakingApplicationsIterator(
	ctx context.Context,
) sharedtypes.RecordIterator[types.Application] {
	unstakingApplicationsStore := k.getApplicationUnstakingStore(ctx)
	applicationStore := k.getApplicationStore(ctx)

	unstakingAppsIterator := storetypes.KVStorePrefixIterator(unstakingApplicationsStore, []byte{})

	applicationAccessor := applicationFromPrimaryKeyAccessorFn(applicationStore, k.cdc)
	return sharedtypes.NewRecordIterator(unstakingAppsIterator, applicationAccessor)
}

// GetAllTransferringApplicationsIterator returns an iterator for all applications
// that are currently transferring.
func (k Keeper) GetAllTransferringApplicationsIterator(
	ctx context.Context,
) sharedtypes.RecordIterator[types.Application] {
	transferApplicationsStore := k.getApplicationTransferStore(ctx)
	applicationStore := k.getApplicationStore(ctx)

	transferringAppsIterator := storetypes.KVStorePrefixIterator(transferApplicationsStore, []byte{})

	applicationAccessor := applicationFromPrimaryKeyAccessorFn(applicationStore, k.cdc)
	return sharedtypes.NewRecordIterator(transferringAppsIterator, applicationAccessor)
}

// GetDelegationsIterator returns an iterator for applications which are currently
// delegated to a specific gateway.
func (k Keeper) GetDelegationsIterator(
	ctx context.Context,
	gatewayAddress string,
) sharedtypes.RecordIterator[types.Application] {
	delegationsStore := k.getDelegationStore(ctx)
	applicationStore := k.getApplicationStore(ctx)

	// Using the gateway address as a prefix key means the iterator will only return
	// entries whose keys begin with this gateway's address. This effectively filters
	// the store to only return delegations related to this specific gateway.
	gatewayKey := types.StringKey(gatewayAddress)
	delegationsIterator := storetypes.KVStorePrefixIterator(delegationsStore, gatewayKey)

	delegationAccessor := applicationFromPrimaryKeyAccessorFn(applicationStore, k.cdc)
	return sharedtypes.NewRecordIterator(delegationsIterator, delegationAccessor)
}

// GetUndelegationsIterator returns an iterator for applications that have pending undelegations.
// If ALL_UNDELEGATIONS is passed as the application address, it will return all pending undelegations.
func (k Keeper) GetUndelegationsIterator(
	ctx context.Context,
	applicationAddress string,
) sharedtypes.RecordIterator[types.Undelegation] {
	undelegationsStore := k.getUndelegationStore(ctx)

	appKey := []byte{}
	if applicationAddress != ALL_UNDELEGATIONS {
		appKey = types.ApplicationKey(applicationAddress)
	}

	// Using the application address as a prefix key means the iterator will only return
	// entries whose keys begin with this application's address. This effectively filters
	// the store to only return undelegations related to this specific application.
	// If ALL_UNDELEGATIONS is used, an empty prefix is passed, returning all undelegations.
	undelegationsIterator := storetypes.KVStorePrefixIterator(undelegationsStore, appKey)

	undelegationAccessor := undelegationAccessorFn(k.cdc)
	return sharedtypes.NewRecordIterator(undelegationsIterator, undelegationAccessor)
}

// applicationFromPrimaryKeyAccessorFn creates a function that retrieves an
// application from its primary key.
//
// This creates a closure that can be used by the RecordIterator to convert primary
// keys into the actual Application objects they reference.
func applicationFromPrimaryKeyAccessorFn(
	applicationStore storetypes.KVStore,
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[types.Application] {
	return func(applicationKey []byte) (types.Application, error) {
		applicationBz := applicationStore.Get(applicationKey)
		if applicationBz == nil {
			return types.Application{}, fmt.Errorf("expected application to exist for key: %v", applicationKey)
		}

		var application types.Application
		cdc.MustUnmarshal(applicationBz, &application)

		return application, nil
	}
}

// undelegationAccessorFn creates a function that retrieves an undelegation record
// from its serialized bytes
//
// Returns an accessor function that takes serialized undelegation bytes and returns
// a deserialized Undelegation object
func undelegationAccessorFn(
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[types.Undelegation] {
	return func(undelegationBz []byte) (types.Undelegation, error) {
		if undelegationBz == nil {
			return types.Undelegation{}, fmt.Errorf("expecting undelegation bytes to be non-nil")
		}

		var undelegation types.Undelegation
		cdc.MustUnmarshal(undelegationBz, &undelegation)

		return undelegation, nil
	}
}

// getApplicationStore returns a KVStore for the application data
func (k Keeper) getApplicationStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
}

// getDelegationStore returns a KVStore for application delegations
func (k Keeper) getDelegationStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.DelegationKeyPrefix))
}

// getUndelegationStore returns a KVStore for application undelegations
func (k Keeper) getUndelegationStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.UndelegationKeyPrefix))
}

// getApplicationUnstakingStore returns a KVStore for unstaking applications
func (k Keeper) getApplicationUnstakingStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationUnstakingKeyPrefix))
}

// getApplicationTransferStore returns a KVStore for application transfers
func (k Keeper) getApplicationTransferStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationTransferKeyPrefix))
}

// GetAllApplicationsIterator returns a RecordIterator over all Application records.
func (k Keeper) GetAllApplicationsIterator(ctx context.Context) sharedtypes.RecordIterator[types.Application] {
	applicationStore := k.getApplicationStore(ctx)
	applicationIterator := storetypes.KVStorePrefixIterator(applicationStore, []byte{})

	applicationUnmarshallerFn := getApplicationAccessorFn(k.cdc, k.logger)
	return sharedtypes.NewRecordIterator(applicationIterator, applicationUnmarshallerFn)
}

// getApplicationAccessorFn constructs a DataRecordAccessor function which:
// 1. Receives a serialized Application value bytes
// 2. Unmarshals it into an Application object
// 3. Initializes any nil fields in the Application object
// Returns:
// - An Application object and an error
func getApplicationAccessorFn(
	cdc codec.BinaryCodec,
	logger log.Logger,
) sharedtypes.DataRecordAccessor[types.Application] {
	return func(applicationBz []byte) (types.Application, error) {
		if applicationBz == nil {
			return types.Application{}, fmt.Errorf("expecting application bytes to be non-nil")
		}

		var application types.Application
		cdc.MustUnmarshal(applicationBz, &application)
		initializeNilApplicationFields(logger, &application)
		return application, nil
	}
}

// initializeNilApplicationFields initializes any nil fields in the application object to their default values.
// - Adding `(gogoproto.nullable)=false` to repeated proto fields acts on the underlying type, not the slice or map type.
// - As a result, slices or maps will be nil if no values are provided in the proto message.
// - This function ensures that the application object has all fields initialized to their default values.
//
// TODO_TECHDEBT: This function is a workaround for the CosmosSDK codec treating empty slices and maps as nil.
// - We should investigate how to make the codec treat empty slices and maps as empty instead of nil.
// - For more context, see: https://github.com/pokt-network/poktroll/pull/1103#discussion_r1992258822
func initializeNilApplicationFields(keeperLogger log.Logger, app *types.Application) {
	logger := keeperLogger.With("module", "application").With("method", "initializeNilApplicationFields")

	if app.ServiceConfigs == nil {
		app.ServiceConfigs = make([]*sharedtypes.ApplicationServiceConfig, 0)
		logger.Warn(fmt.Sprintf("should never happen: app.ServiceConfigs was nil, initializing to empty slice for app %s", app.Address))
	}

	// The CosmosSDK codec treats empty slices and maps as nil, so we need to
	// ensure that they are initialized as empty.
	if app.PendingUndelegations == nil {
		app.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
	}
}
