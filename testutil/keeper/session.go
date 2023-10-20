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

var (
	TestServiceId = "svc1"

	TestAppAddress  = sample.AccAddress()
	TestApplication = apptypes.Application{
		Address: TestAppAddress,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		ServiceIds: []*sharedtypes.ServiceId{
			{
				Id: TestServiceId,
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
				ServiceId: &sharedtypes.ServiceId{Id: TestServiceId},
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     TestSupplierUrl,
						RpcType: sharedtypes.RPCType_JSON_RPC,
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

	mockAppKeeper := appKeeperMock(t)
	mockSupplierKeeper := supplierKeeperMock(t)

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

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

func appKeeperMock(t testing.TB) types.ApplicationKeeper {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockAppKeeper := mocks.NewMockApplicationKeeper(ctrl)
	mockAppKeeper.EXPECT().GetApplication(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, appAddr string) (apptypes.Application, bool) {
			if appAddr == TestAppAddress {
				return TestApplication, true
			}
			return apptypes.Application{}, false
		},
	)
	mockAppKeeper.EXPECT().GetApplication(gomock.Any(), TestAppAddress).AnyTimes().Return(TestApplication, true)
	return mockAppKeeper
}

func supplierKeeperMock(t testing.TB) types.SupplierKeeper {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockSupplierKeeper := mocks.NewMockSupplierKeeper(ctrl)
	mockSupplierKeeper.EXPECT().GetAllSupplier(gomock.Any()).AnyTimes().Return([]sharedtypes.Supplier{TestSupplier})
	return mockSupplierKeeper
}
