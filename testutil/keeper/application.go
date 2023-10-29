package keeper

import (
	"fmt"
	"testing"

	"cosmossdk.io/depinject"
	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mocks "pocket/testutil/application/mocks"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
	gatewaytypes "pocket/x/gateway/types"
)

var AddrToPubKeyMap map[string]cryptotypes.PubKey

func init() {
	AddrToPubKeyMap = make(map[string]cryptotypes.PubKey)
}

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
	// because the gateways are not staked this is needed to mock the GetPubKey method which
	// returns nil if the actor cannot be found on chain (ie. not staked)
	mockAccountKeeper.EXPECT().GetPubKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, address sdk.AccAddress) (cryptotypes.PubKey, error) {
			addr := address.String()
			found, ok := AddrToPubKeyMap[addr]
			if !ok {
				return nil, fmt.Errorf("public key not found for address: %s", addr)
			}
			return found, nil
		},
	).AnyTimes()

	mockGatewayKeeper := mocks.NewMockGatewayKeeper(ctrl)
	mockGatewayKeeper.EXPECT().GetGateway(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, addr string) (gatewaytypes.Gateway, bool) {
			stake := sdk.NewCoin("upokt", sdk.NewInt(10000))
			return gatewaytypes.Gateway{
				Address: addr,
				Stake:   &stake,
			}, true
		},
	).AnyTimes()

	applicationDeps := depinject.Supply(mockGatewayKeeper)

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
		applicationDeps,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
