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

	"github.com/pokt-network/poktroll/testutil/supplier/mocks"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SupplierModuleKeepers is a struct that contains the keepers needed for testing
// the supplier module.
type SupplierModuleKeepers struct {
	*keeper.Keeper
	types.SharedKeeper
	// Tracks the amount of funds returned to the supplier owner when the supplier is unbonded.
	SupplierUnstakedFundsMap map[string]int64
}

func SupplierKeeper(t testing.TB) (SupplierModuleKeepers, context.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	logger := log.NewTestLogger(t)
	sdkCtx := sdk.NewContext(stateStore, cmtproto.Header{}, false, logger)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Set a simple map to track the where the supplier stake is returned when
	// the supplier is unbonded.
	supplierUnstakedFundsMap := make(map[string]int64)

	ctrl := gomock.NewController(t)
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), types.ModuleName, gomock.Any()).AnyTimes()
	mockBankKeeper.EXPECT().SpendableCoins(gomock.Any(), gomock.Any()).AnyTimes()
	mockBankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).AnyTimes().
		Do(func(ctx context.Context, module string, addr sdk.AccAddress, coins sdk.Coins) {
			supplierUnstakedFundsMap[addr.String()] += coins[0].Amount.Int64()
		})

	// Construct a real shared keeper.
	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		logger,
		authority.String(),
	)
	require.NoError(t, sharedKeeper.SetParams(sdkCtx, sharedtypes.DefaultParams()))

	serviceKeeper := servicekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
	)

	supplierKeeper := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		sharedKeeper,
		serviceKeeper,
	)

	// Initialize params
	require.NoError(t, supplierKeeper.SetParams(sdkCtx, types.DefaultParams()))

	// Move block height to 1 to get a non zero session end height
	ctx := SetBlockHeight(sdkCtx, 1)

	// Add existing services used in the test.
	serviceKeeper.SetService(ctx, sharedtypes.Service{Id: "svcId"})
	serviceKeeper.SetService(ctx, sharedtypes.Service{Id: "svcId2"})

	supplierModuleKeepers := SupplierModuleKeepers{
		Keeper:                   &supplierKeeper,
		SharedKeeper:             sharedKeeper,
		SupplierUnstakedFundsMap: supplierUnstakedFundsMap,
	}

	return supplierModuleKeepers, ctx
}
