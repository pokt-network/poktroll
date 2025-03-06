package keeper

import (
	"context"
	"fmt"
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
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/migration/mocks"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// MigrationKeeperConfig is a configuration struct for the MigrationKeeper testutil.
type MigrationKeeperConfig struct {
	bankKeeper migrationtypes.BankKeeper
}

// MigrationKeeperOptionFn is a function which receives and potentially modifies
// the MigrationKeeperConfig during construction of the MigrationKeeper testutil.
type MigrationKeeperOptionFn func(cfg *MigrationKeeperConfig)

// MigrationKeeper returns a new migration module keeper with mocked dependencies
// (i.e. gateway, app, & supplier keepers). Mocked dependencies are configurable
// via the MigrationKeeperOptionFns.
func MigrationKeeper(
	t testing.TB,
	opts ...MigrationKeeperOptionFn,
) (keeper.Keeper, cosmostypes.Context) {
	storeKey := storetypes.NewKVStoreKey(migrationtypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ctrl := gomock.NewController(t)
	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)

	cfg := defaultConfigWithMocks(ctrl)
	for _, opt := range opts {
		opt(cfg)
	}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockAccountKeeper,
		cfg.bankKeeper,
	)

	ctx := cosmostypes.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, migrationtypes.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}

// WithBankKeeper assigns the given BankKeeper to the MigrationKeeperConfig.
func WithBankKeeper(bankKeeper migrationtypes.BankKeeper) MigrationKeeperOptionFn {
	return func(cfg *MigrationKeeperConfig) {
		cfg.bankKeeper = bankKeeper
	}
}

// defaultConfigWithMocks returns a MigrationKeeperConfig with a mocked bank keeper
// which respond the following methods by updating mapAccAddrCoins accordingly:
// - SpendableCoins
// - MintCoins
// - SendCoinsFromModuleToAccount
func defaultConfigWithMocks(ctrl *gomock.Controller) *MigrationKeeperConfig {
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().
		SpendableCoins(gomock.Any(), gomock.Any()).
		DoAndReturn(mockBankKeeperSpendableCoins).AnyTimes()
	mockBankKeeper.EXPECT().
		MintCoins(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(mockBankKeeperMintCoins).AnyTimes()
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(mockBankKeeperSendFromModuleToAccount).AnyTimes()

	return &MigrationKeeperConfig{
		bankKeeper: mockBankKeeper,
	}
}

// mockBankKeeperSpendableCoins implements a static version of the corresponding bank
// keeper method that interacts with an in-memory map of account addresses to balances.
func mockBankKeeperSpendableCoins(_ context.Context, addr cosmostypes.AccAddress) cosmostypes.Coins {
	mapMu.RLock()
	defer mapMu.RUnlock()
	if coins, ok := mapAccAddrCoins[addr.String()]; ok {
		return coins
	}
	return cosmostypes.Coins{}
}

// mockBankKeeperMintCoins implements a static version of the corresponding bank
// keeper method that interacts with an in-memory map of account addresses to balances.
func mockBankKeeperMintCoins(
	_ context.Context,
	moduleName string,
	mintCoins cosmostypes.Coins) error {
	mapMu.Lock()
	defer mapMu.Unlock()
	moduleAddr := authtypes.NewModuleAddress(moduleName)

	// Check for an existing balance
	balance, ok := mapAccAddrCoins[moduleAddr.String()]
	if !ok {
		balance = cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0))
	}

	balance = balance.Add(mintCoins...)

	// Update the balance.
	mapAccAddrCoins[moduleAddr.String()] = balance

	return nil
}

// mockBankKeeperSendFromModuleToAccount implements a static version of the corresponding
// bank keeper method that interacts with an in-memory map of account addresses to balances.
func mockBankKeeperSendFromModuleToAccount(
	_ context.Context,
	senderModule string,
	recipientAddr cosmostypes.AccAddress,
	sendCoins cosmostypes.Coins,
) error {
	mapMu.Lock()
	defer mapMu.Unlock()
	moduleAddr := authtypes.NewModuleAddress(senderModule)

	moduleBalance, ok := mapAccAddrCoins[moduleAddr.String()]
	if !ok {
		return fmt.Errorf("no module account for %s (address %s)", senderModule, moduleAddr)
	}

	remainingModuleBalance, isNegative := moduleBalance.SafeSub(sendCoins...)
	if isNegative {
		return fmt.Errorf("not enough coins to send (%s) from module account %q", sendCoins, senderModule)
	}

	recipientBalance, ok := mapAccAddrCoins[recipientAddr.String()]
	if !ok {
		recipientBalance = cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0))
	}

	recipientBalance = recipientBalance.Add(sendCoins...)

	mapAccAddrCoins[moduleAddr.String()] = remainingModuleBalance
	mapAccAddrCoins[recipientAddr.String()] = recipientBalance

	return nil
}
