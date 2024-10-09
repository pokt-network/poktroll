package keeper

import (
	"context"
	"sync"
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

func ServiceKeeper(t testing.TB) (keeper.Keeper, context.Context) {
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
	mockBankKeeper.EXPECT().
		SpendableCoins(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
				mapMu.RLock()
				defer mapMu.RUnlock()
				if coins, ok := mapAccAddrCoins[addr.String()]; ok {
					return coins
				}
				return sdk.Coins{}
			},
		).AnyTimes()
	mockBankKeeper.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
				mapMu.Lock()
				defer mapMu.Unlock()
				coins := mapAccAddrCoins[senderAddr.String()]
				if coins.AmountOf("upokt").GT(amt.AmountOf("upokt")) {
					mapAccAddrCoins[senderAddr.String()] = coins.Sub(amt...)
					return nil
				}
				return types.ErrServiceNotEnoughFunds
			},
		).AnyTimes()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

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
	coins := sdk.NewCoins(sdk.Coin{Denom: denom, Amount: math.NewIntFromUint64(amount)})
	mapMu.Lock()
	defer mapMu.Unlock()
	mapAccAddrCoins[addrBech32.String()] = coins
}
