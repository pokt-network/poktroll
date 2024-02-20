package keeper

import (
	"context"
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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/proof/mocks"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func ProofKeeper(
	t testing.TB,
	sessionByAppAddr proof.SessionsByAppAddress,
) (keeper.Keeper, context.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ctrl := gomock.NewController(t)
	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockSessionKeeper.EXPECT().
		GetSession(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(
				ctx context.Context,
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

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockSessionKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
