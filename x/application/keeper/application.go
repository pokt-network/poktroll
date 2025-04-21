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
func (k Keeper) SetApplication(ctx context.Context, application types.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	appBz := k.cdc.MustMarshal(&application)
	store.Set(types.ApplicationKey(application.Address), appBz)
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx context.Context,
	appAddr string,
) (app types.Application, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))

	appBz := store.Get(types.ApplicationKey(appAddr))
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

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(ctx context.Context, appAddr string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(appAddr))
}

// GetAllApplications returns all application
func (k Keeper) GetAllApplications(ctx context.Context) (apps []types.Application) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

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

// GetAllApplicationsIterator returns an iterator over all application records.
// GetAllApplicationsIterator returns a RecordIterator over all Application records.
func (k Keeper) GetAllApplicationsIterator(ctx context.Context) sharedtypes.RecordIterator[*types.Application] {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ApplicationKeyPrefix))
	applicationIterator := storetypes.KVStorePrefixIterator(store, []byte{})

	applicationUnmarshallerFn := getApplicationAccessorFn(k.cdc)
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
) sharedtypes.DataRecordAccessor[*types.Application] {
	return func(applicationBz []byte) (*types.Application, error) {
		if applicationBz == nil {
			return nil, nil
		}

		var application types.Application
		cdc.MustUnmarshal(applicationBz, &application)
		initializeNilApplicationFields(k.logger, &application)
		return &application, nil
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
