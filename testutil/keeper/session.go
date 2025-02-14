package keeper

import (
	"context"
	"encoding/hex"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/session/mocks"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	TestServiceId1  = "svc1"  // staked for by app1 & supplier1
	TestServiceId11 = "svc11" // staked for by app1

	TestServiceId2  = "svc2"  // staked for by app2 & supplier1
	TestServiceId22 = "svc22" // staked for by app2

	TestServiceId12 = "svc12" // staked for by app1, app2 & supplier1

	TestApp1Address = "pokt1mdccn4u38eyjdxkk4h0jaddw4n3c72u82m5m9e" // Generated via sample.AccAddress()
	TestApp1        = apptypes.Application{
		Address: TestApp1Address,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: TestServiceId1,
			},
			{
				ServiceId: TestServiceId11,
			},
			{
				ServiceId: TestServiceId12,
			},
		},
	}

	TestApp2Address = "pokt133amv5suh75zwkxxcq896azvmmwszg99grvk9f" // Generated via sample.AccAddress()
	TestApp2        = apptypes.Application{
		Address: TestApp1Address,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: TestServiceId2,
			},
			{
				ServiceId: TestServiceId22,
			},
			{
				ServiceId: TestServiceId12,
			},
		},
	}

	TestSupplierUrl             = "http://olshansky.info"
	TestSupplierOperatorAddress = sample.AccAddress()
	TestSupplier                = sharedtypes.Supplier{
		OperatorAddress: TestSupplierOperatorAddress,
		Stake:           &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: TestServiceId1,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
			{
				ServiceId: TestServiceId2,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_GRPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
			{
				ServiceId: TestServiceId12,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_GRPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
		},
	}
)

// keeperConfig is a configuration struct to be used during keeper construction
// to modify its behavior.
type keeperConfig struct {
	sharedParams *sharedtypes.Params
}

// KeeperOptionFn is a function type that sets/updates fields on the keeperConfig.
type KeeperOptionFn func(*keeperConfig)

// WithSharedModuleParams returns a KeeperOptionFn that sets the moduleParams field
// on the keeperConfig.
func WithSharedModuleParams(sharedParams *sharedtypes.Params) KeeperOptionFn {
	return func(c *keeperConfig) {
		c.sharedParams = sharedParams
	}
}

func SessionKeeper(t testing.TB, opts ...KeeperOptionFn) (keeper.Keeper, context.Context) {
	t.Helper()

	cfg := &keeperConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ctrl := gomock.NewController(t)
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)

	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)
	mockAccountKeeper.EXPECT().GetAccount(gomock.Any(), gomock.Any()).AnyTimes()

	mockAppKeeper := defaultAppKeeperMock(t)
	mockSupplierKeeper := defaultSupplierKeeperMock(t)

	sharedParams := sharedtypes.DefaultParams()
	if cfg.sharedParams != nil {
		sharedParams = *cfg.sharedParams
	}
	mockSharedKeeper := defaultSharedKeeperMock(t, &sharedParams)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockAccountKeeper,
		mockBankKeeper,
		mockAppKeeper,
		mockSupplierKeeper,
		mockSharedKeeper,
	)

	// TODO_TECHDEBT: See the comment at the bottom of this file explaining
	// why we don't support options yet.
	// for _, opt := range opts {
	// 	opt(k)
	// }

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	// In prod, the hashes of all block heights are stored in the hash store while
	// the block hashes below are hardcoded to match the hardcoded session IDs used
	// in the `session_hydrator_test.go`.
	// TODO_IMPROVE: Use fixtures populated by block hashes and their corresponding
	// session IDs for each block height in the [0, N] interval, instead of using
	// in-place hardcoded values.
	// Store block hashes to be used in tests
	blockHash := map[int64]string{
		0: "",                                                                 // no session at block height 0
		1: "1b1051b7bf236fea13efa65b6be678514fa5b6ea0ae9a7a4b68d45f95e4f18e0", // 1st session
		5: "261594ddc3c8afc5b4c63f59ee58e89d3a115bcd164c83fd79349de0b1ffd21d", // 2nd session
		9: "251665c7cf286a30fbd98acd983c63e9a34efc16496511373405e24eb02a8fb9", // 3rd session
	}

	storeAdapter := runtime.KVStoreAdapter(runtime.NewKVStoreService(storeKey).OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.BlockHashKeyPrefix))
	for height, hash := range blockHash {
		hashBz, err := hex.DecodeString(hash)
		require.NoError(t, err)
		store.Set(types.BlockHashKey(height), hashBz)
	}

	return k, ctx
}

func defaultAppKeeperMock(t testing.TB) types.ApplicationKeeper {
	t.Helper()
	ctrl := gomock.NewController(t)

	getAppFn := func(_ context.Context, appAddr string) (apptypes.Application, bool) {
		switch appAddr {
		case TestApp1Address:
			return TestApp1, true
		case TestApp2Address:
			return TestApp2, true
		default:
			return apptypes.Application{}, false
		}
	}

	mockAppKeeper := mocks.NewMockApplicationKeeper(ctrl)
	mockAppKeeper.EXPECT().GetApplication(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(getAppFn)
	mockAppKeeper.EXPECT().GetApplication(gomock.Any(), TestApp1Address).AnyTimes().Return(TestApp1, true)

	return mockAppKeeper
}

func defaultSupplierKeeperMock(t testing.TB) types.SupplierKeeper {
	t.Helper()
	ctrl := gomock.NewController(t)

	allSuppliers := []sharedtypes.Supplier{TestSupplier}

	mockSupplierKeeper := mocks.NewMockSupplierKeeper(ctrl)
	mockSupplierKeeper.EXPECT().GetAllSuppliers(gomock.Any()).AnyTimes().Return(allSuppliers)

	return mockSupplierKeeper
}

func defaultSharedKeeperMock(t testing.TB, params *sharedtypes.Params) types.SharedKeeper {
	t.Helper()
	ctrl := gomock.NewController(t)

	if params == nil {
		params = new(sharedtypes.Params)
		*params = sharedtypes.DefaultParams()
	}

	mockSharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	mockSharedKeeper.EXPECT().GetParams(gomock.Any()).
		Return(*params).
		AnyTimes()
	return mockSharedKeeper
}
