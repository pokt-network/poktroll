package keeper

// ┌──────────────────────────────────────────────────────────────────────────────────────────┐
// │ 📦  Application Primary Store                                                           │
// ├──────────────────────────────────────────────────────────────────────────────────────────┤
// │ Store (bucket)        Key (prefix + addr)                 → Value                       │
// │──────────────────────────────────────────────────────────────────────────────────────────│
// │ applicationStore      AK                                  → appBz                       │
// └──────────────────────────────────────────────────────────────────────────────────────────┘
//
// Legend
//   AK (ApplicationKey) : types.ApplicationKey(appAddr)
//                         = "Application/address/" || appAddr.
//   appBz               : protobuf-marshaled types.Application.
//
// Fast-path look-up
//   • AppAddr → applicationStore → appBz.
//
// Index counts
//   ① Primary data (one record per Application)

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

// SetApplication sets an application in the store and updates all related indexes.
// - Indexes the application in all relevant indexes
// - Stores the application in the main application store
func (k Keeper) SetApplication(ctx context.Context, application types.Application) {
	// Index the application in all relevant indexes
	k.indexApplicationUnstaking(ctx, application)
	k.indexApplicationTransfer(ctx, application)
	k.indexApplicationDelegations(ctx, application)
	k.indexApplicationUndelegations(ctx, application)

	// Store the application
	applicationStore := k.getApplicationStore(ctx)
	appBz := k.cdc.MustMarshal(&application)
	applicationStore.Set(types.ApplicationKey(application.Address), appBz)
}

// GetApplication retrieves an application by address.
// - Returns false if not found
// - Initializes PendingUndelegations as an empty map if nil
// - Initializes DelegateeGatewayAddresses as an empty slice if nil
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

// RemoveApplication deletes an application from the store and all related indexes.
// - Removes from unstaking, transfer, undelegation, and delegation indexes
// - Deletes from the main application store
func (k Keeper) RemoveApplication(ctx context.Context, application types.Application) {
	// Remove the application from all relevant indexes
	k.removeApplicationUnstakingIndex(ctx, application.Address)
	k.removeApplicationTransferIndex(ctx, application.Address)
	k.removeApplicationUndelegationIndexes(ctx, application.Address)
	k.removeApplicationDelegationsIndexes(ctx, application)

	// Remove the application from the store
	applicationStore := k.getApplicationStore(ctx)
	applicationStore.Delete(types.ApplicationKey(application.Address))
}

// GetAllApplications returns all applications in the store.
// - Ensures PendingUndelegations is always initialized
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

// GetAllUnstakingApplicationsIterator returns an iterator over all unstaking applications.
// - Uses unstaking applications store as the source of truth
// - Accesses full application objects via primary key accessor
func (k Keeper) GetAllUnstakingApplicationsIterator(
	ctx context.Context,
) sharedtypes.RecordIterator[types.Application] {
	unstakingApplicationsStore := k.getApplicationUnstakingStore(ctx)
	applicationStore := k.getApplicationStore(ctx)

	unstakingAppsIterator := storetypes.KVStorePrefixIterator(unstakingApplicationsStore, []byte{})

	applicationAccessor := applicationFromPrimaryKeyAccessorFn(applicationStore, k.cdc)
	return sharedtypes.NewRecordIterator(unstakingAppsIterator, applicationAccessor)
}

// GetAllTransferringApplicationsIterator returns an iterator over all transferring applications.
// - Uses transferring applications store as the source of truth
// - Accesses full application objects via primary key accessor
func (k Keeper) GetAllTransferringApplicationsIterator(
	ctx context.Context,
) sharedtypes.RecordIterator[types.Application] {
	transferApplicationsStore := k.getApplicationTransferStore(ctx)
	applicationStore := k.getApplicationStore(ctx)

	transferringAppsIterator := storetypes.KVStorePrefixIterator(transferApplicationsStore, []byte{})

	applicationAccessor := applicationFromPrimaryKeyAccessorFn(applicationStore, k.cdc)
	return sharedtypes.NewRecordIterator(transferringAppsIterator, applicationAccessor)
}

// GetDelegationsIterator returns an iterator for applications delegated to a specific gateway.
// - Filters delegations by gateway address prefix
// - Returns only delegations related to the given gateway
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

// GetUndelegationsIterator returns an iterator for applications with pending undelegations.
// - If ALL_UNDELEGATIONS is passed, returns all pending undelegations
// - Otherwise, filters by application address prefix
func (k Keeper) GetUndelegationsIterator(
	ctx context.Context,
	applicationAddress string,
) sharedtypes.RecordIterator[types.PendingUndelegation] {
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

// applicationFromPrimaryKeyAccessorFn creates a DataRecordAccessor for Applications.
// - Retrieves an application from its primary key in the store
// - Returns an error if the application does not exist
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

// undelegationAccessorFn creates a DataRecordAccessor for Undelegations.
// - Deserializes undelegation bytes into an Undelegation object
// - Returns an error if bytes are nil
func undelegationAccessorFn(
	cdc codec.BinaryCodec,
) sharedtypes.DataRecordAccessor[types.PendingUndelegation] {
	return func(undelegationBz []byte) (types.PendingUndelegation, error) {
		if undelegationBz == nil {
			return types.PendingUndelegation{}, fmt.Errorf("expecting undelegation bytes to be non-nil")
		}

		var undelegation types.PendingUndelegation
		cdc.MustUnmarshal(undelegationBz, &undelegation)

		return undelegation, nil
	}
}

// getApplicationStore returns a prefixed KVStore for application data.
func (k Keeper) getApplicationStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
}

// getDelegationStore returns a prefixed KVStore for application delegations.
func (k Keeper) getDelegationStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.DelegationKeyPrefix))
}

// getUndelegationStore returns a prefixed KVStore for application undelegations.
func (k Keeper) getUndelegationStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.UndelegationKeyPrefix))
}

// getApplicationUnstakingStore returns a prefixed KVStore for unstaking applications.
func (k Keeper) getApplicationUnstakingStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationUnstakingKeyPrefix))
}

// getApplicationTransferStore returns a prefixed KVStore for application transfers.
func (k Keeper) getApplicationTransferStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationTransferKeyPrefix))
}

// GetAllApplicationsIterator returns a RecordIterator over all Application records.
// - Uses the main application store and unmarshals each record
// - Initializes nil fields in each application object
func (k Keeper) GetAllApplicationsIterator(ctx context.Context) sharedtypes.RecordIterator[types.Application] {
	applicationStore := k.getApplicationStore(ctx)
	applicationIterator := storetypes.KVStorePrefixIterator(applicationStore, []byte{})

	applicationUnmarshallerFn := getApplicationAccessorFn(k.cdc, k.logger)
	return sharedtypes.NewRecordIterator(applicationIterator, applicationUnmarshallerFn)
}

// getApplicationAccessorFn constructs a DataRecordAccessor for Applications.
// - Receives serialized Application bytes
// - Unmarshals into an Application object
// - Initializes nil fields in the Application object
// - Returns the Application object and an error
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

// initializeNilApplicationFields initializes nil fields in the Application object to default values.
// - Ensures ServiceConfigs is always a non-nil slice
// - Ensures PendingUndelegations is always a non-nil map
// - Logs a warning if ServiceConfigs was nil (should not happen)
// - Workaround for CosmosSDK codec treating empty slices/maps as nil
// - See: https://github.com/pokt-network/poktroll/pull/1103#discussion_r1992258822
// - TODO_TECHDEBT: Investigate making the codec treat empty slices/maps as empty instead of nil
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
