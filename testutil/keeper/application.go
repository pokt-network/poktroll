package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/application/mocks"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// stakedGatewayMap is used to mock whether a gateway is staked or not for use
// in the application's mocked gateway keeper. This enables the tester to
// control whether a gateway is "staked" or not and whether it can be delegated to
// WARNING: Using this map may cause issues if running multiple tests in parallel
var stakedGatewayMap = make(map[string]struct{})

// ApplicationModuleKeepers is a struct that contains the keepers needed for testing
// the application module.
type ApplicationModuleKeepers struct {
	*keeper.Keeper
	types.SharedKeeper
}

// NewApplicationModuleKeepers creates a new application keeper for testing along
// with its dependencies then returns the application module keepers and context.
func NewApplicationModuleKeepers(t testing.TB) (ApplicationModuleKeepers, context.Context) {
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
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).AnyTimes()
	mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).AnyTimes()

	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)
	mockAccountKeeper.EXPECT().GetAccount(gomock.Any(), gomock.Any()).AnyTimes()

	mockGatewayKeeper := mocks.NewMockGatewayKeeper(ctrl)
	mockGatewayKeeper.EXPECT().GetGateway(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, addr string) (gatewaytypes.Gateway, bool) {
			if _, ok := stakedGatewayMap[addr]; !ok {
				return gatewaytypes.Gateway{}, false
			}
			stake := sdk.NewCoin("upokt", math.NewInt(10000))
			return gatewaytypes.Gateway{
				Address: addr,
				Stake:   &stake,
			}, true
		},
	).AnyTimes()

	mockSharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	mockSharedKeeper.EXPECT().GetParams(gomock.Any()).
		DoAndReturn(func(_ context.Context) sharedtypes.Params {
			return sharedtypes.DefaultParams()
		}).
		AnyTimes()
	mockSharedKeeper.EXPECT().GetSessionEndHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, queryHeight int64) int64 {
			return testsession.GetSessionEndHeightWithDefaultParams(queryHeight)
		}).
		AnyTimes()

	appKeeper := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockAccountKeeper,
		mockGatewayKeeper,
		mockSharedKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	require.NoError(t, appKeeper.SetParams(ctx, types.DefaultParams()))

	// Move block height to 1 to get a non zero session end height
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithBlockHeight(1)

	applicationModuleKeepers := ApplicationModuleKeepers{
		Keeper:       &appKeeper,
		SharedKeeper: mockSharedKeeper,
	}

	return applicationModuleKeepers, ctx
}

// ApplicationKeeper creates a new application keeper for testing and returns
// the keeper and context.
func ApplicationKeeper(t testing.TB) (keeper.Keeper, context.Context) {
	t.Helper()

	applicationModuleKeepers, ctx := NewApplicationModuleKeepers(t)

	return *applicationModuleKeepers.Keeper, ctx
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
