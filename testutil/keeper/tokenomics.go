package keeper

import (
	"context"
	"math"
	"testing"

	"cosmossdk.io/log"
	cosmosmath "cosmossdk.io/math"
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
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/app/volatile"
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
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_IN_THIS_COMMIT: godoc...
var defaultMintConfigs = []ModuleBalanceConfig{
	{
		ModuleName: suppliertypes.ModuleName,
		Coins:      cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000000000)),
	},
	{
		ModuleName: apptypes.ModuleName,
		Coins:      cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000000000)),
	},
}

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

	Codec codec.Codec
}

// tokenomicsModuleKeepersConfig is a configuration struct for a TokenomicsModuleKeepers
// instance. Its fields are intended to be set/updated by TokenomicsModuleKeepersOptFn
// functions which are passed during integration construction.
type tokenomicsModuleKeepersConfig struct {
	registry          codectypes.InterfaceRegistry
	tokenLogicModules []tlm.TokenLogicModule
	initKeepersFns    []func(context.Context, *TokenomicsModuleKeepers) context.Context
	// moduleParams is a map of module names to their respective module parameters.
	// This is used to set the initial module parameters in the keeper.
	moduleParams map[string]cosmostypes.Msg

	moduleBalances []ModuleBalanceConfig
}

// TODO_IN_THIS_COMMIT: godoc...
type ModuleBalanceConfig struct {
	ModuleName string
	Coins      cosmostypes.Coins
}

// TokenomicsModuleKeepersOptFn is a function which receives and potentially modifies
// the context and tokenomics keepers during construction of the aggregation.
type TokenomicsModuleKeepersOptFn func(cfg *tokenomicsModuleKeepersConfig)

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
		Stake:          &cosmostypes.Coin{Denom: "upokt", Amount: cosmosmath.NewInt(100000)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}

	// Prepare the test supplier.
	supplierOwnerAddr := sample.AccAddress()
	supplier := sharedtypes.Supplier{
		OwnerAddress:    supplierOwnerAddr,
		OperatorAddress: supplierOwnerAddr,
		Stake:           &cosmostypes.Coin{Denom: "upokt", Amount: cosmosmath.NewInt(100000)},
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: service.Id,
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            supplierOwnerAddr,
						RevSharePercentage: 100,
					},
				},
			},
		},
	}

	sdkCtx := cosmostypes.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

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
	mockApplicationKeeper.EXPECT().
		UnbondApplication(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()
	mockApplicationKeeper.EXPECT().
		EndBlockerUnbondApplications(gomock.Any()).
		Return(nil).
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
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), tokenomicstypes.ModuleName, suppliertypes.ModuleName, gomock.Any()).
		AnyTimes()
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), apptypes.ModuleName, tokenomicstypes.ModuleName, gomock.Any()).
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
	mockSharedKeeper.EXPECT().
		GetParams(gomock.Any()).
		Return(sharedtypes.DefaultParams()).
		AnyTimes()

	// Mock the session keeper
	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockSessionKeeper.EXPECT().
		GetParams(gomock.Any()).
		Return(sessiontypes.DefaultParams()).
		AnyTimes()

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

	relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(sdkCtx, log.NewNopLogger(), service.Id, servicekeeper.TargetNumRelays)
	mockServiceKeeper.EXPECT().
		GetRelayMiningDifficulty(gomock.Any(), gomock.Any()).
		Return(relayMiningDifficulty, true).
		AnyTimes()

	tokenLogicModules := tlm.NewDefaultTokenLogicModules()

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
		tokenLogicModules,
	)

	// Add a block proposer address to the context
	sdkCtx = sdkCtx.WithProposer(sample.ConsAddress())

	// Initialize params
	require.NoError(t, k.SetParams(sdkCtx, tokenomicstypes.DefaultParams()))

	return k, sdkCtx, application.Address, supplier.OperatorAddress, service
}

// NewTokenomicsModuleKeepers is a helper function to create a tokenomics keeper
// and a context. It uses real dependencies for all upstream keepers.
func NewTokenomicsModuleKeepers(
	t testing.TB,
	logger log.Logger,
	opts ...TokenomicsModuleKeepersOptFn,
) (_ TokenomicsModuleKeepers, ctx context.Context) {
	t.Helper()

	cfg := &tokenomicsModuleKeepersConfig{
		tokenLogicModules: tlm.NewDefaultTokenLogicModules(),
		moduleBalances:    defaultMintConfigs,
	}
	for _, opt := range opts {
		opt(cfg)
	}

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
	ctx = cosmostypes.NewContext(stateStore, cmtproto.Header{}, false, logger)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Add a block proposer address to the context
	sdkCtx = sdkCtx.WithProposer(sample.ConsAddress())

	// Prepare the account keeper.
	registry := codectypes.NewInterfaceRegistry()
	if cfg.registry != nil {
		registry = cfg.registry
	}
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

	for _, moduleBalanceCfg := range cfg.moduleBalances {
		err := bankKeeper.MintCoins(sdkCtx, moduleBalanceCfg.ModuleName, moduleBalanceCfg.Coins)
		require.NoError(t, err)
	}

	// Construct a real shared keeper.
	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[sharedtypes.StoreKey]),
		logger,
		authority.String(),
	)
	require.NoError(t, sharedKeeper.SetParams(sdkCtx, sharedtypes.DefaultParams()))

	if params, ok := cfg.moduleParams[sharedtypes.ModuleName]; ok {
		err := sharedKeeper.SetParams(ctx, *params.(*sharedtypes.Params))
		require.NoError(t, err)
	}

	// Construct gateway keeper with a mocked bank keeper.
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[gatewaytypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		sharedKeeper,
	)
	require.NoError(t, gatewayKeeper.SetParams(sdkCtx, gatewaytypes.DefaultParams()))

	if params, ok := cfg.moduleParams[gatewaytypes.ModuleName]; ok {
		err := gatewayKeeper.SetParams(ctx, *params.(*gatewaytypes.Params))
		require.NoError(t, err)
	}

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

	if params, ok := cfg.moduleParams[apptypes.ModuleName]; ok {
		err := appKeeper.SetParams(ctx, *params.(*apptypes.Params))
		require.NoError(t, err)
	}

	// Construct a service keeper needed by the supplier keeper.
	serviceKeeper := servicekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[servicetypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
	)

	if params, ok := cfg.moduleParams[servicetypes.ModuleName]; ok {
		err := serviceKeeper.SetParams(ctx, *params.(*servicetypes.Params))
		require.NoError(t, err)
	}

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

	if params, ok := cfg.moduleParams[suppliertypes.ModuleName]; ok {
		err := supplierKeeper.SetParams(ctx, *params.(*suppliertypes.Params))
		require.NoError(t, err)
	}

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

	if params, ok := cfg.moduleParams[sessiontypes.ModuleName]; ok {
		err := sessionKeeper.SetParams(ctx, *params.(*sessiontypes.Params))
		require.NoError(t, err)
	}

	// Construct a real proof keeper so that claims & proofs can be created.
	proofKeeper := proofkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[prooftypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		sessionKeeper,
		appKeeper,
		accountKeeper,
		sharedKeeper,
		serviceKeeper,
	)
	require.NoError(t, proofKeeper.SetParams(sdkCtx, prooftypes.DefaultParams()))

	if params, ok := cfg.moduleParams[prooftypes.ModuleName]; ok {
		err := proofKeeper.SetParams(ctx, *params.(*prooftypes.Params))
		require.NoError(t, err)
	}

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
		cfg.tokenLogicModules,
	)

	require.NoError(t, tokenomicsKeeper.SetParams(sdkCtx, tokenomicstypes.DefaultParams()))

	if params, ok := cfg.moduleParams[tokenomicstypes.ModuleName]; ok {
		err := tokenomicsKeeper.SetParams(ctx, *params.(*tokenomicstypes.Params))
		require.NoError(t, err)
	}

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
	for _, fn := range cfg.initKeepersFns {
		ctx = fn(ctx, &keepers)
	}

	return keepers, ctx
}

// WithService is an option to set the service in the tokenomics module keepers.
func WithService(service sharedtypes.Service) TokenomicsModuleKeepersOptFn {
	setService := func(ctx context.Context, keepers *TokenomicsModuleKeepers) context.Context {
		keepers.SetService(ctx, service)
		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setService)
	}
}

// WithApplication is an option to set the application in the tokenomics module keepers.
func WithApplication(applicaion apptypes.Application) TokenomicsModuleKeepersOptFn {
	setApp := func(ctx context.Context, keepers *TokenomicsModuleKeepers) context.Context {
		keepers.SetApplication(ctx, applicaion)
		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setApp)
	}
}

// WithSupplier is an option to set the supplier in the tokenomics module keepers.
func WithSupplier(supplier sharedtypes.Supplier) TokenomicsModuleKeepersOptFn {
	setSupplier := func(ctx context.Context, keepers *TokenomicsModuleKeepers) context.Context {
		keepers.SetSupplier(ctx, supplier)
		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setSupplier)
	}
}

// WithProposerAddr is an option to set the proposer address in the context used
// by the tokenomics module keepers.
func WithProposerAddr(addr string) TokenomicsModuleKeepersOptFn {
	setProposerAddr := func(ctx context.Context, keepers *TokenomicsModuleKeepers) context.Context {
		consAddr, err := cosmostypes.ConsAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		sdkCtx = sdkCtx.WithProposer(consAddr)
		return sdkCtx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setProposerAddr)
	}
}

// WithTokenLogicModules returns a TokenomicsModuleKeepersOptFn that sets the given
// TLM processors on the tokenomicsModuleKeepersConfig.
func WithTokenLogicModules(processors []tlm.TokenLogicModule) TokenomicsModuleKeepersOptFn {
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.tokenLogicModules = processors
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func WithDaoRewardBech32(daoRewardAddr string) TokenomicsModuleKeepersOptFn {
	tokenomicsParams := tokenomicstypes.DefaultParams()
	tokenomicsParams.DaoRewardAddress = daoRewardAddr

	return WithModuleParams(map[string]cosmostypes.Msg{
		tokenomicstypes.ModuleName: &tokenomicsParams,
	})
}

// WithModuleParams returns a KeeperOptionFn that sets the moduleParams field
// on the keeperConfig.
func WithModuleParams(moduleParams map[string]cosmostypes.Msg) TokenomicsModuleKeepersOptFn {
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.moduleParams = moduleParams
	}
}

// WithProofRequirement is an option to enable or disable the proof requirement
// in the tokenomics module keepers by setting the proof request probability to
// 1 or 0, respectively whie setting the proof requirement threshold to 0 or
// MaxInt64, respectively.
func WithProofRequirement(proofRequirementReason prooftypes.ProofRequirementReason) TokenomicsModuleKeepersOptFn {
	setProofRequirement := func(ctx context.Context, keepers *TokenomicsModuleKeepers) context.Context {
		proofParams := keepers.ProofKeeper.GetParams(ctx)

		// By default, NEVER require a proof (via neither probabilistic nor threshold).
		proofParams.ProofRequestProbability = 0
		proofRequirementThreshold := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
		proofParams.ProofRequirementThreshold = &proofRequirementThreshold

		// Override the default proof requirement based on proofRequirementReason.
		switch proofRequirementReason {
		case prooftypes.ProofRequirementReason_NOT_REQUIRED:
		case prooftypes.ProofRequirementReason_PROBABILISTIC:
			// Require a proof 100% of the time probabilistically.
			proofParams.ProofRequestProbability = 1
		case prooftypes.ProofRequirementReason_THRESHOLD:
			// Require a proof of any claim amount (i.e. anything greater than 0).
			proofRequirementThreshold := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
			proofParams.ProofRequirementThreshold = &proofRequirementThreshold
		}

		if err := keepers.ProofKeeper.SetParams(ctx, proofParams); err != nil {
			panic(err)
		}

		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setProofRequirement)
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func WithRegistry(registry codectypes.InterfaceRegistry) TokenomicsModuleKeepersOptFn {
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.registry = registry
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func WithModuleBalances(mints []ModuleBalanceConfig) TokenomicsModuleKeepersOptFn {
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.moduleBalances = mints
	}
}
