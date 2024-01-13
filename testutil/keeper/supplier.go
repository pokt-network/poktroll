package keeper

import (
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

	"github.com/pokt-network/poktroll/testutil/supplier"
	"github.com/pokt-network/poktroll/testutil/supplier/mocks"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SupplierKeeper returns a mocked supplier keeper and context for testing.
func SupplierKeeper(
	t testing.TB,
	sessionByAppAddr supplier.SessionsByAppAddress,
) (*keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	ctrl := gomock.NewController(t)
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().
		DelegateCoinsFromAccountToModule(
			gomock.Any(),
			gomock.Any(),
			types.ModuleName,
			gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		UndelegateCoinsFromModuleToAccount(
			gomock.Any(),
			types.ModuleName,
			gomock.Any(),
			gomock.Any()).
		AnyTimes()

	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockSessionKeeper.EXPECT().
		GetSession(gomock.AssignableToTypeOf(sdk.Context{}), gomock.Any()).
		DoAndReturn(
			func(
				ctx sdk.Context,
				req *sessiontypes.QueryGetSessionRequest,
			) (*sessiontypes.QueryGetSessionResponse, error) {
				session, ok := sessionByAppAddr[req.GetApplicationAddress()]
				require.Truef(
					t,
					ok,
					"application address not provided during mock construction: %q",
					req.ApplicationAddress,
				)

				return &sessiontypes.QueryGetSessionResponse{
					Session: &sessiontypes.Session{
						Header: &sessiontypes.SessionHeader{
							ApplicationAddress:      session.GetApplication().GetAddress(),
							Service:                 req.GetService(),
							SessionStartBlockHeight: 1,
							SessionId:               session.GetSessionId(),
							SessionEndBlockHeight:   5,
						},
						SessionId:           session.GetSessionId(),
						SessionNumber:       1,
						NumBlocksPerSession: session.GetNumBlocksPerSession(),
						Application:         session.GetApplication(),
						Suppliers:           session.GetSuppliers(),
					},
				}, nil
			},
		).AnyTimes()

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"SupplierParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,

		mockBankKeeper,
	)
	k.SupplySessionKeeper(mockSessionKeeper)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
