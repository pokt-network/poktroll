package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	math "cosmossdk.io/math"
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

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_TECHDEBT: Replace `AnyTimes` w/ `Times/MinTimes/MaxTimes` as the tests
// mature to be explicit about the number of expected tests.

func TokenomicsKeeper(t testing.TB) (
	k keeper.Keeper,
	ttx context.Context,
	appAddr string,
	supplierAddr string,
) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	// Initialize the in-memory database.
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	// Initialize the codec and other necessary components.
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// The on-chain governance address.
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the test application.
	application := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100000)},
	}

	// Prepare the test supplier.
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100000)},
	}

	ctrl := gomock.NewController(t)

	// Mock the application keeper.
	mockApplicationKeeper := mocks.NewMockApplicationKeeper(ctrl)

	// Get test application if the address matches.
	mockApplicationKeeper.EXPECT().
		GetApplication(gomock.Any(), gomock.Eq(application.Address)).
		Return(application, true).
		AnyTimes()

	// Get zero-value application if the address does not match.
	mockApplicationKeeper.EXPECT().
		GetApplication(gomock.Any(), gomock.Not(application.Address)).
		Return(apptypes.Application{}, false).
		AnyTimes()

	// Mock SetApplication.
	mockApplicationKeeper.EXPECT().
		SetApplication(gomock.Any(), gomock.Any()).
		AnyTimes()

	// Get test supplier if the address matches.
	mockSupplierKeeper := mocks.NewMockSupplierKeeper(ctrl)
	mockSupplierKeeper.EXPECT().
		GetSupplier(gomock.Any(), supplier.Address).
		Return(supplier, true).
		AnyTimes()

	// Mock the bank keeper.
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().
		MintCoins(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), suppliertypes.ModuleName, gomock.Any(), gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), apptypes.ModuleName, gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		UndelegateCoinsFromModuleToAccount(gomock.Any(), apptypes.ModuleName, gomock.Any(), gomock.Any()).
		AnyTimes()

	// Mock the account keeper
	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)
	mockAccountKeeper.EXPECT().GetAccount(gomock.Any(), gomock.Any()).AnyTimes()

	tokenomicsKeeper := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockAccountKeeper,
		mockApplicationKeeper,
		mockSupplierKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	require.NoError(t, tokenomicsKeeper.SetParams(ctx, types.DefaultParams()))

	return tokenomicsKeeper, ctx, application.Address, supplier.Address
}
