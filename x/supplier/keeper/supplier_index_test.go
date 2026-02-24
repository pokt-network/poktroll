package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// newMinimalKeeper creates a minimal Keeper with only the fields needed for
// index operations. This avoids circular imports with testutil/keeper.
func newMinimalKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	sdkCtx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	k := Keeper{
		cdc:          cdc,
		storeService: runtime.NewKVStoreService(storeKey),
		logger:       log.NewNopLogger(),
	}

	return k, sdkCtx
}

func TestGetSupplierServiceConfigUpdates_SkipsOrphanedIndexEntries(t *testing.T) {
	k, ctx := newMinimalKeeper(t)

	operatorAddr := "pokt1testoperator"
	serviceId := "svc-test"

	serviceConfig := &sharedtypes.ServiceConfigUpdate{
		OperatorAddress: operatorAddr,
		Service: &sharedtypes.SupplierServiceConfig{
			ServiceId: serviceId,
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://localhost:8080",
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: make([]*sharedtypes.ConfigOption, 0),
				},
			},
		},
		ActivationHeight:   1,
		DeactivationHeight: sharedtypes.NoDeactivationHeight,
	}

	// Create a supplier with the service config and index it.
	supplier := sharedtypes.Supplier{
		OperatorAddress:      operatorAddr,
		ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{serviceConfig},
	}
	k.indexSupplierServiceConfigUpdates(ctx, supplier)

	// Sanity check: verify the config is retrievable.
	configs := k.getSupplierServiceConfigUpdates(ctx, operatorAddr, "")
	require.Len(t, configs, 1)
	require.NotNil(t, configs[0].Service)
	require.Equal(t, serviceId, configs[0].Service.ServiceId)

	// Create an orphaned index entry by deleting only the primary record
	// while leaving the supplier→primary-key index entry intact.
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)
	primaryKey := types.ServiceConfigUpdateKey(*serviceConfig)
	serviceConfigUpdateStore.Delete(primaryKey)

	// Verify that getSupplierServiceConfigUpdates skips the orphaned entry
	// instead of panicking or returning a zero-value entry with Service == nil.
	configs = k.getSupplierServiceConfigUpdates(ctx, operatorAddr, "")
	require.Empty(t, configs, "orphaned index entries should be skipped")

	// Also verify filtering by service ID handles orphaned entries.
	configs = k.getSupplierServiceConfigUpdates(ctx, operatorAddr, serviceId)
	require.Empty(t, configs, "orphaned index entries should be skipped when filtering by service ID")
}

func TestGetSupplierServiceConfigUpdates_MixedValidAndOrphaned(t *testing.T) {
	k, ctx := newMinimalKeeper(t)

	operatorAddr := "pokt1testoperator"

	// Create two service configs for the same supplier.
	svcConfig1 := &sharedtypes.ServiceConfigUpdate{
		OperatorAddress: operatorAddr,
		Service: &sharedtypes.SupplierServiceConfig{
			ServiceId: "svc1",
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{Url: "http://localhost:8081", RpcType: sharedtypes.RPCType_JSON_RPC},
			},
		},
		ActivationHeight:   1,
		DeactivationHeight: sharedtypes.NoDeactivationHeight,
	}
	svcConfig2 := &sharedtypes.ServiceConfigUpdate{
		OperatorAddress: operatorAddr,
		Service: &sharedtypes.SupplierServiceConfig{
			ServiceId: "svc2",
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{Url: "http://localhost:8082", RpcType: sharedtypes.RPCType_JSON_RPC},
			},
		},
		ActivationHeight:   1,
		DeactivationHeight: sharedtypes.NoDeactivationHeight,
	}

	supplier := sharedtypes.Supplier{
		OperatorAddress:      operatorAddr,
		ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{svcConfig1, svcConfig2},
	}
	k.indexSupplierServiceConfigUpdates(ctx, supplier)

	// Delete only the first service config's primary entry to create one orphan.
	serviceConfigUpdateStore := k.getServiceConfigUpdatesStore(ctx)
	primaryKey1 := types.ServiceConfigUpdateKey(*svcConfig1)
	serviceConfigUpdateStore.Delete(primaryKey1)

	// Should return only the valid (non-orphaned) config.
	configs := k.getSupplierServiceConfigUpdates(ctx, operatorAddr, "")
	require.Len(t, configs, 1)
	require.Equal(t, "svc2", configs[0].Service.ServiceId)
}
