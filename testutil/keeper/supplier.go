package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/supplier"
	"github.com/pokt-network/poktroll/testutil/supplier/mocks"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func SupplierKeeper(t testing.TB, sessionByAppAddr supplier.SessionsByAppAddress) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	ctrl := gomock.NewController(t)
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().DelegateCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).AnyTimes()
	mockBankKeeper.EXPECT().UndelegateCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).AnyTimes()

	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockSessionKeeper.EXPECT().
		GetSession(gomock.AssignableToTypeOf(sdk.Context{}), gomock.Any()).
		DoAndReturn(
			func(
				ctx sdk.Context,
				req *sessiontypes.QueryGetSessionRequest,
			) (*sessiontypes.QueryGetSessionResponse, error) {
				session, ok := sessionByAppAddr[req.GetApplicationAddress()]
				require.Truef(t, ok, "application address not provided during mock construction: %q", req.ApplicationAddress)

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

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockSessionKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
