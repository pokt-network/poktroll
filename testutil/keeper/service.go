package keeper

import (
	"sync"
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

	"github.com/pokt-network/poktroll/testutil/service/mocks"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
)

var (
	// mapAccAddrCoins is used by the mock BankModule to determine who has what
	// coins, if they are sufficient to pay the fee for adding a service.
	mapAccAddrCoins = make(map[string]sdk.Coins)
	mapMu           = sync.RWMutex{}
)

// ServiceKeeper returns an instance of the keeper for the service module
// with a mocked dependency of the BankModule, this is used for testing purposes.
func ServiceKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
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
		SpendableCoins(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
			mapMu.RLock()
			defer mapMu.RUnlock()
			if coins, ok := mapAccAddrCoins[addr.String()]; ok {
				return coins
			}
			return sdk.Coins{}
		}).
		AnyTimes()
	mockBankKeeper.EXPECT().
		DelegateCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
			mapMu.Lock()
			defer mapMu.Unlock()
			coins := mapAccAddrCoins[senderAddr.String()]
			if coins.AmountOf("upokt").GT(amt.AmountOf("upokt")) {
				mapAccAddrCoins[senderAddr.String()] = coins.Sub(amt...)
				return nil
			}
			return types.ErrServiceNotEnoughFunds
		}).
		AnyTimes()

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"ServiceParams",
	)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		mockBankKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

// AddAccToAccMapCoins adds to the mapAccAddrCoins map the coins specified as
// parameters, to the function under the address specified. When it cleans up
// it deletes the entry in the map for the provided address.
func AddAccToAccMapCoins(t *testing.T, addr, denom string, amount uint64) {
	t.Helper()
	t.Cleanup(func() {
		mapMu.Lock()
		delete(mapAccAddrCoins, addr)
		mapMu.Unlock()
	})
	addrBech32, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	coins := sdk.NewCoins(sdk.Coin{Denom: denom, Amount: sdk.NewIntFromUint64(amount)})
	mapMu.Lock()
	defer mapMu.Unlock()
	mapAccAddrCoins[addrBech32.String()] = coins
}

// RemoveFromAccMapCoins removes an address from the mapAccAddrCoins map
func RemoveFromAccMapCoins(t *testing.T, addr string) {
	t.Helper()
	mapMu.Lock()
	defer mapMu.Unlock()
	delete(mapAccAddrCoins, addr)
}
