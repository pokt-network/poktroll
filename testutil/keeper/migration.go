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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

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
) (keeper.Keeper, sdk.Context) {
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

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

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

func defaultConfigWithMocks(ctrl *gomock.Controller) *MigrationKeeperConfig {
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
		MintCoins(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()

	return &MigrationKeeperConfig{
		bankKeeper: mockBankKeeper,
	}
}
