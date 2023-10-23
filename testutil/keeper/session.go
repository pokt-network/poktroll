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
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
	mocks "pocket/testutil/session/mocks"
	apptypes "pocket/x/application/types"
	"pocket/x/session/keeper"
	"pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

type option[V any] func(k *keeper.Keeper)

var (
	TestServiceId1 = "svc1"
	TestServiceId2 = "svc2"

	TestApp1Address = "pokt106grzmkmep67pdfrm6ccl9snynryjqus6l3vct" // Generated via sample.AccAddress()
	TestApp1        = apptypes.Application{
		Address: TestApp1Address,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		ServiceIds: []*sharedtypes.ServiceId{
			{
				Id: TestServiceId1,
			},
			{
				Id: TestServiceId2,
			},
		},
	}

	TestApp2Address = "pokt1dm7tr0a99ja232gzt5rjtrl7hj6z6h40669fwh" // Generated via sample.AccAddress()
	TestApp2        = apptypes.Application{
		Address: TestApp1Address,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		ServiceIds: []*sharedtypes.ServiceId{
			{
				Id: TestServiceId1,
			},
			{
				Id: TestServiceId2,
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
				ServiceId: &sharedtypes.ServiceId{Id: TestServiceId1},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_JSON_RPC,
					},
				},
			},
			{
				ServiceId: &sharedtypes.ServiceId{Id: TestServiceId2},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_GRPC,
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
