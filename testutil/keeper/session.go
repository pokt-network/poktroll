package keeper

import (
	"context"
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/golang/mock/gomock"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/session/mocks"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/stretchr/testify/require"
)

type option[V any] func(k *keeper.Keeper)

var (
	TestServiceId1  = "svc1"  // staked for by app1 & supplier1
	TestServiceId11 = "svc11" // staked for by app1

	TestServiceId2  = "svc2"  // staked for by app2 & supplier1
	TestServiceId22 = "svc22" // staked for by app2

	TestServiceId12 = "svc12" // staked for by app1, app2 & supplier1

	TestApp1Address = "pokt1mdccn4u38eyjdxkk4h0jaddw4n3c72u82m5m9e" // Generated via sample.AccAddress()
	TestApp1        = apptypes.Application{
		Address: TestApp1Address,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: TestServiceId1},
			},
			{
				Service: &sharedtypes.Service{Id: TestServiceId11},
			},
			{
				Service: &sharedtypes.Service{Id: TestServiceId12},
			},
		},
	}

	TestApp2Address = "pokt133amv5suh75zwkxxcq896azvmmwszg99grvk9f" // Generated via sample.AccAddress()
	TestApp2        = apptypes.Application{
		Address: TestApp1Address,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: TestServiceId2},
			},
			{
				Service: &sharedtypes.Service{Id: TestServiceId22},
			},
			{
				Service: &sharedtypes.Service{Id: TestServiceId12},
			},
		},
	}

	TestSupplierUrl     = "http://olshansky.info"
	TestSupplierAddress = sample.AccAddress()
	TestSupplier        = sharedtypes.Supplier{
		Address: TestSupplierAddress,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: &sharedtypes.Service{Id: TestServiceId1},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
			{
				Service: &sharedtypes.Service{Id: TestServiceId2},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_GRPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
			{
				Service: &sharedtypes.Service{Id: TestServiceId12},
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

func SessionKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockAppKeeper := defaultAppKeeperMock(t)
	mockSupplierKeeper := defaultSupplierKeeperMock(t)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"SessionParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,

		mockAppKeeper,
		mockSupplierKeeper,
	)

	// TODO_TECHDEBT: See the comment at the bottom of this file explaining
	// why we don't support options yet.
	// for _, opt := range opts {
	// 	opt(k)
	// }

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

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
	mockSupplierKeeper.EXPECT().GetAllSupplier(gomock.Any()).AnyTimes().Return(allSuppliers)

	return mockSupplierKeeper
}

// TODO_TECHDEBT: Figure out how to vary the supplierKeep on a per test basis with exposing `SupplierKeeper publically`

// type option[V any] func(k *keeper.Keeper)

// WithPublisher returns an option function which sets the given publishCh of the
// resulting observable when passed to NewObservable().
// func WithSupplierKeeperMock(supplierKeeper types.SupplierKeeper) option[any] {
// 	return func(k *keeper.Keeper) {
// 		k.supplierKeeper = supplierKeeper
// 	}
// }
