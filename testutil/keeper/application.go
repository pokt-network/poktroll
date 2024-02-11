package keeper

import (
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
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

	mocks "github.com/pokt-network/poktroll/testutil/application/mocks"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// stakedGatewayMap is used to mock whether a gateway is staked or not for use
// in the application's mocked gateway keeper. This enables the tester to
// control whether a gateway is "staked" or not and whether it can be delegated to
// WARNING: Using this map may cause issues if running multiple tests in parallel
var stakedGatewayMap = make(map[string]struct{})

func ApplicationKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
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
	mockBankKeeper.EXPECT().DelegateCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).AnyTimes()
	mockBankKeeper.EXPECT().UndelegateCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).AnyTimes()

	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)
	mockAccountKeeper.EXPECT().GetAccount(gomock.Any(), gomock.Any()).AnyTimes()

	mockGatewayKeeper := mocks.NewMockGatewayKeeper(ctrl)
	mockGatewayKeeper.EXPECT().GetGateway(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, addr string) (gatewaytypes.Gateway, bool) {
			if _, ok := stakedGatewayMap[addr]; !ok {
				return gatewaytypes.Gateway{}, false
			}
			stake := sdk.NewCoin("upokt", sdkmath.NewInt(10000))
			return gatewaytypes.Gateway{
				Address: addr,
				Stake:   &stake,
			}, true
		},
	).AnyTimes()

	// TODO_CONSOLIDATE: This was passed-in instead of authority.String() in the
	// original code. It's not clear what the difference is.
	// paramsSubspace := typesparams.NewSubspace(cdc,
	// 	types.Amino,
	// 	storeKey,
	// 	memStoreKey,
	// 	"ApplicationParams",
	// )
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockAccountKeeper,
		mockGatewayKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

// AddGatewayToStakedGatewayMap adds the given gateway address to the staked
// gateway map for use in the application's mocked gateway keeper and ensures
// that it is removed from the map when the test is complete
func AddGatewayToStakedGatewayMap(t *testing.T, gatewayAddr string) {
	t.Helper()
	stakedGatewayMap[gatewayAddr] = struct{}{}
	t.Cleanup(func() {
		delete(stakedGatewayMap, gatewayAddr)
	})
}

// RemoveGatewayFromStakedGatewayMap removes the given gateway address from the
// staked gateway map for use in the application's mocked gateway keeper
func RemoveGatewayFromStakedGatewayMap(t *testing.T, gatewayAddr string) {
	t.Helper()
	delete(stakedGatewayMap, gatewayAddr)
}
