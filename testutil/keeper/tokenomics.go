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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	mocks "github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TokenomicsKeeper(t testing.TB) (
	k *keeper.Keeper, s sdk.Context,
	appAddr, supplierAddr string,
) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	// Initialize the in-memory tendermint database
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	// Initialize the codec and other necessary components
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	ctrl := gomock.NewController(t)

	// The on-chain governance address
	authority := authtypes.NewModuleAddress("gov").String()

	// Prepare the test application
	application := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100000)},
	}

	// Prepare the test supplier
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100000)},
	}

	// Mock the application keeper
	mockApplicationKeeper := mocks.NewMockApplicationKeeper(ctrl)
	mockApplicationKeeper.EXPECT().
		GetApplication(gomock.Any(), gomock.Eq(application.Address)).
		Return(application, true).
		AnyTimes()
	mockApplicationKeeper.EXPECT().
		GetApplication(gomock.Any(), gomock.Not(application.Address)).
		Return(apptypes.Application{}, false).
		AnyTimes()
	mockApplicationKeeper.EXPECT().
		SetApplication(gomock.Any(), gomock.Any()).
		AnyTimes()

	// Mock the supplier keeper
	mockSupplierKeeper := mocks.NewMockSupplierKeeper(ctrl)
	mockSupplierKeeper.EXPECT().
		GetSupplier(gomock.Any(), supplier.Address).
		Return(supplier, true).
		AnyTimes()

	// Mock the bank keeper
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

	// Initialize the tokenomics keeper
	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"TokenomicsParams",
	)
	tokenomicsKeeper := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,

		mockBankKeeper,
		mockApplicationKeeper,
		mockSupplierKeeper,

		authority,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	tokenomicsKeeper.SetParams(ctx, types.DefaultParams())

	return tokenomicsKeeper, ctx, application.Address, supplier.Address
}
