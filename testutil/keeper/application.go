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

// ApplicationKeeper returns a mocked application keeper and context for testing
// it mocks the chain having staked gateways via the use of the stakedGatewayMap
func ApplicationKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
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
			stake := sdk.NewCoin("upokt", sdk.NewInt(10000))
			return gatewaytypes.Gateway{
				Address: addr,
				Stake:   &stake,
			}, true
		},
	).AnyTimes()

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"ApplicationParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		mockBankKeeper,
		mockAccountKeeper,
		mockGatewayKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

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
