package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TokenomicsModuleKeepers is an aggregation of the tokenomics keeper, all its dependency
// keepers, and the codec that they share. Each keeper is embedded such that the
// TokenomicsModuleKeepers implements all the interfaces of the keepers.
// To call a method which is common to multiple keepers (e.g. `#SetParams()`),
// the field corresponding to the desired keeper on which to call the method
// MUST be specified (e.g. `keepers.AccountKeeper#SetParams()`).
type TokenomicsModuleKeepers struct {
	*tokenomicskeeper.Keeper
	tokenomicstypes.AccountKeeper
	tokenomicstypes.BankKeeper
	tokenomicstypes.ApplicationKeeper
	tokenomicstypes.SupplierKeeper
	tokenomicstypes.ProofKeeper
	tokenomicstypes.SharedKeeper
	tokenomicstypes.SessionKeeper
	tokenomicstypes.ServiceKeeper

	Codec *codec.ProtoCodec
}

// TokenomicsKeepersOpt is a function which receives and potentially modifies the context
// and tokenomics keepers during construction of the aggregation.
type TokenomicsModuleKeepersOpt func(context.Context, *TokenomicsModuleKeepers) context.Context

func TokenomicsKeeper(t testing.TB) (tokenomicsKeeper tokenomicskeeper.Keeper, ctx context.Context) {
	t.Helper()
	k, ctx, _, _, _ := TokenomicsKeeperWithActorAddrs(t)
	return k, ctx
}

// TODO_TECHDEBT: Remove this and force everyone to use NewTokenomicsModuleKeepers.
// There is a difference in the method signatures and mocking, which was simply
// a result of the evolution of the testutil package.
// TODO_REFACTOR(@Olshansk): Rather than making `service`, `appAddr` and `supplierOperatorAddr`
// explicit params, make them passable by the caller as options.
func TokenomicsKeeperWithActorAddrs(t testing.TB) (
	tokenomicsKeeper tokenomicskeeper.Keeper,
	ctx context.Context,
	appAddr string,
	supplierOperatorAddr string,
	service *sharedtypes.Service,
) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(tokenomicstypes.StoreKey)

	service = &sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}

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
		Address:        sample.AccAddress(),
		Stake:          &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100000)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{Service: service}},
	}

	// Prepare the test supplier.
	supplierOwnerAddr := sample.AccAddress()
	supplier := sharedtypes.Supplier{
		OwnerAddress:    supplierOwnerAddr,
		OperatorAddress: supplierOwnerAddr,
		Stake:           &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100000)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: service,
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            supplierOwnerAddr,
						RevSharePercentage: 100,
					},
				},
			},
		},
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

	// Mock the supplier keeper.
	mockSupplierKeeper := mocks.NewMockSupplierKeeper(ctrl)
	// Mock SetSupplier.
	mockSupplierKeeper.EXPECT().
		SetSupplier(gomock.Any(), gomock.Any()).
		AnyTimes()

	// Get test supplier if the address matches.
	mockSupplierKeeper.EXPECT().
		GetSupplier(gomock.Any(), gomock.Eq(supplier.OperatorAddress)).
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
		SendCoinsFromModuleToAccount(gomock.Any(), tokenomicstypes.ModuleName, gomock.Any(), gomock.Any()).
		AnyTimes()

	// Mock the account keeper
	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)
	mockAccountKeeper.EXPECT().GetAccount(gomock.Any(), gomock.Any()).AnyTimes()

	// Mock the proof keeper
	mockProofKeeper := mocks.NewMockProofKeeper(ctrl)
	mockProofKeeper.EXPECT().GetAllClaims(gomock.Any()).AnyTimes()

	// Mock the shared keeper
	mockSharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	mockSharedKeeper.EXPECT().GetProofWindowCloseHeight(gomock.Any(), gomock.Any()).AnyTimes()

	// Mock the session keeper
	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)

	// Mock the service keeper
	mockServiceKeeper := mocks.NewMockServiceKeeper(ctrl)

	mockServiceKeeper.EXPECT().
		GetService(gomock.Any(), gomock.Eq(service.Id)).
		Return(*service, true).
		AnyTimes()
	mockServiceKeeper.EXPECT().
		GetService(gomock.Any(), gomock.Any()).
		Return(sharedtypes.Service{}, false).
		AnyTimes()

	k := tokenomicskeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockAccountKeeper,
		mockApplicationKeeper,
		mockSupplierKeeper,
		mockProofKeeper,
		mockSharedKeeper,
		mockSessionKeeper,
		mockServiceKeeper,
	)

	sdkCtx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Add a block proposer address to the context
	valAddr, err := cosmostypes.ValAddressFromBech32(sample.ConsAddress())
	require.NoError(t, err)
	consensusAddr := cosmostypes.ConsAddress(valAddr)
	sdkCtx = sdkCtx.WithProposer(consensusAddr)

	// Initialize params
	require.NoError(t, k.SetParams(sdkCtx, tokenomicstypes.DefaultParams()))

	return k, sdkCtx, application.Address, supplier.OperatorAddress, service
}

// NewTokenomicsModuleKeepers is a helper function to create a tokenomics keeper
// and a context. It uses real dependencies for all upstream keepers.
func NewTokenomicsModuleKeepers(
	t testing.TB,
	logger log.Logger,
	opts ...TokenomicsModuleKeepersOpt,
) (_ TokenomicsModuleKeepers, ctx context.Context) {
	t.Helper()

	// Collect store keys for all keepers which be constructed & interact with the state store.
	keys := storetypes.NewKVStoreKeys(
		tokenomicstypes.StoreKey,
		banktypes.StoreKey,
		gatewaytypes.StoreKey,
		authtypes.StoreKey,
		sessiontypes.StoreKey,
		apptypes.StoreKey,
		suppliertypes.StoreKey,
		prooftypes.StoreKey,
		sharedtypes.StoreKey,
		servicetypes.StoreKey,
	)

	// Construct a multistore & mount store keys for each keeper that will interact with the state store.
	stateStore := integration.CreateMultiStore(keys, log.NewNopLogger())

	// Use the test logger by default (i.e. if none is given).
	if logger == nil {
		logger = log.NewTestLogger(t)
	}

	// Prepare the context
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, logger)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Add a block proposer address to the context
	valAddr, err := cosmostypes.ValAddressFromBech32(sample.ConsAddress())
	require.NoError(t, err)
	consensusAddr := cosmostypes.ConsAddress(valAddr)
	sdkCtx = sdkCtx.WithProposer(consensusAddr)

	// ctx.SetAccount
	// Prepare the account keeper.
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)

	// Prepare the chain's authority
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Construct a real account keeper so that public keys can be queried.
	addrCodec := addresscodec.NewBech32Codec(app.AccountAddressPrefix)
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		// These module accounts are necessary in order to settle balances
		// during claim expiration.
		map[string][]string{
			minttypes.ModuleName:       {authtypes.Minter},
			suppliertypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
			apptypes.ModuleName:        {authtypes.Minter, authtypes.Burner},
			tokenomicstypes.ModuleName: {authtypes.Minter, authtypes.Burner},
		},
		addrCodec,
		app.AccountAddressPrefix,
		authority.String(),
	)

	// Construct a real bank keeper so that the balances can be updated & verified
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accountKeeper,
		make(map[string]bool),
		authority.String(),
		logger,
	)
	require.NoError(t, bankKeeper.SetParams(sdkCtx, banktypes.DefaultParams()))

	// Provide some initial funds to the suppliers & applications module accounts.
	err = bankKeeper.MintCoins(sdkCtx, suppliertypes.ModuleName, sdk.NewCoins(sdk.NewCoin("upokt", math.NewInt(1000000000000))))
	require.NoError(t, err)
	err = bankKeeper.MintCoins(sdkCtx, apptypes.ModuleName, sdk.NewCoins(sdk.NewCoin("upokt", math.NewInt(1000000000000))))
	require.NoError(t, err)

	// Construct a real shared keeper.
	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[sharedtypes.StoreKey]),
		logger,
		authority.String(),
	)
	require.NoError(t, sharedKeeper.SetParams(sdkCtx, sharedtypes.DefaultParams()))

	// Construct gateway keeper with a mocked bank keeper.
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[gatewaytypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
	)
	require.NoError(t, gatewayKeeper.SetParams(sdkCtx, gatewaytypes.DefaultParams()))

	// Construct an application keeper to add apps to sessions.
	appKeeper := appkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		gatewayKeeper,
		sharedKeeper,
	)
	require.NoError(t, appKeeper.SetParams(sdkCtx, apptypes.DefaultParams()))

	// Construct a service keeper needed by the supplier keeper.
	serviceKeeper := servicekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[servicetypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
	)

	// Construct a real supplier keeper to add suppliers to sessions.
	supplierKeeper := supplierkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[suppliertypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		sharedKeeper,
		serviceKeeper,
	)
	require.NoError(t, supplierKeeper.SetParams(sdkCtx, suppliertypes.DefaultParams()))

	// Construct a real session keeper so that sessions can be queried.
	sessionKeeper := sessionkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[sessiontypes.StoreKey]),
		logger,
		authority.String(),
		accountKeeper,
		bankKeeper,
		appKeeper,
		supplierKeeper,
		sharedKeeper,
	)
	require.NoError(t, sessionKeeper.SetParams(sdkCtx, sessiontypes.DefaultParams()))

	// Construct a real proof keeper so that claims & proofs can be created.
	proofKeeper := proofkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[prooftypes.StoreKey]),
		logger,
		authority.String(),
		sessionKeeper,
		appKeeper,
		accountKeeper,
		sharedKeeper,
	)
	require.NoError(t, proofKeeper.SetParams(sdkCtx, prooftypes.DefaultParams()))

	// Construct a real tokenomics keeper so that claims & tokenomics can be created.
	tokenomicsKeeper := tokenomicskeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[tokenomicstypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		appKeeper,
		supplierKeeper,
		proofKeeper,
		sharedKeeper,
		sessionKeeper,
		serviceKeeper,
	)

	require.NoError(t, tokenomicsKeeper.SetParams(sdkCtx, tokenomicstypes.DefaultParams()))

	keepers := TokenomicsModuleKeepers{
		Keeper:            &tokenomicsKeeper,
		AccountKeeper:     &accountKeeper,
		BankKeeper:        &bankKeeper,
		ApplicationKeeper: &appKeeper,
		SupplierKeeper:    &supplierKeeper,
		ProofKeeper:       &proofKeeper,
		SharedKeeper:      &sharedKeeper,
		SessionKeeper:     &sessionKeeper,
		ServiceKeeper:     &serviceKeeper,

		Codec: cdc,
	}

	// Apply any options to update the keepers or context prior to returning them.
	ctx = sdkCtx
	for _, opt := range opts {
		ctx = opt(ctx, &keepers)
	}

	return keepers, ctx
}

func WithService(service sharedtypes.Service) TokenomicsModuleKeepersOpt {
	return func(ctx context.Context, keepers *TokenomicsModuleKeepers) context.Context {
		keepers.SetService(ctx, service)
		return ctx
	}
}
