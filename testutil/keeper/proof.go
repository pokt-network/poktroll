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
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/testutil/proof/mocks"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

type ProofKeeperWithDeps struct {
	ProofKeeper       *keeper.Keeper
	SessionKeeper     prooftypes.SessionKeeper
	SupplierKeeper    prooftypes.SupplierKeeper
	ApplicationKeeper prooftypes.ApplicationKeeper
}

func ProofKeeper(t testing.TB) (keeper.Keeper, context.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ctrl := gomock.NewController(t)
	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockAppKeeper := mocks.NewMockApplicationKeeper(ctrl)
	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)

	k, err := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockSessionKeeper,
		mockAppKeeper,
		mockAccountKeeper,
	)
	require.NoError(t, err)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	return k, ctx
}

func NewProofKeeperWithDeps(t testing.TB) (ProofKeeperWithDeps, context.Context) {
	t.Helper()

	// Collect store keys for all keepers which be constructed & interact with the state store.
	keys := storetypes.NewKVStoreKeys(
		types.StoreKey,
		sessiontypes.StoreKey,
		suppliertypes.StoreKey,
		apptypes.StoreKey,
		gatewaytypes.StoreKey,
	)

	// Construct a multistore & mount store keys for each keeper that will interact with the state store.
	stateStore := integration.CreateMultiStore(keys, log.NewNopLogger())

	logger := log.NewTestLogger(t)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, logger)

	// Set block height to 1 so there is a valid session on-chain.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithBlockHeight(1)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Mock the bank keeper.
	ctrl := gomock.NewController(t)
	bankKeeperMock := mocks.NewMockBankKeeper(ctrl)

	// Construct a real account keeper so that public keys can be queried.
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		map[string][]string{minttypes.ModuleName: {authtypes.Minter}},
		addresscodec.NewBech32Codec(app.AccountAddressPrefix),
		app.AccountAddressPrefix,
		authority.String(),
	)

	// Construct gateway keeper with a mocked bank keeper.
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[gatewaytypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeperMock,
	)
	require.NoError(t, gatewayKeeper.SetParams(ctx, gatewaytypes.DefaultParams()))

	// Construct an application keeper to add apps to sessions.
	appKeeper := appkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeperMock,
		accountKeeper,
		gatewayKeeper,
	)
	require.NoError(t, appKeeper.SetParams(ctx, apptypes.DefaultParams()))

	// Construct a real supplier keeper to add suppliers to sessions.
	supplierKeeper := supplierkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[suppliertypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeperMock,
	)
	require.NoError(t, supplierKeeper.SetParams(ctx, suppliertypes.DefaultParams()))

	// Construct a real session keeper so that sessions can be queried.
	sessionKeeper := sessionkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[sessiontypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		bankKeeperMock,
		appKeeper,
		supplierKeeper,
	)
	require.NoError(t, sessionKeeper.SetParams(ctx, sessiontypes.DefaultParams()))

	proofKeeper, err := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[types.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		sessionKeeper,
		appKeeper,
		accountKeeper,
	)
	require.NoError(t, err)
	require.NoError(t, proofKeeper.SetParams(ctx, types.DefaultParams()))

	keeperWithDeps := ProofKeeperWithDeps{
		ProofKeeper:       &proofKeeper,
		SessionKeeper:     &sessionKeeper,
		SupplierKeeper:    &supplierKeeper,
		ApplicationKeeper: &appKeeper,
	}

	return keeperWithDeps, ctx
}
