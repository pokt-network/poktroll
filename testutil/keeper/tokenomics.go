package keeper

import (
	"context"
	"fmt"
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
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	migrationkeeper "github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
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
	tokenomicstypes.StakingKeeper
	tokenomicstypes.DistributionKeeper
	tokenomicstypes.MigrationKeeper

	Codec *codec.ProtoCodec
}

// tokenomicsModuleKeepersConfig is a configuration struct for a TokenomicsModuleKeepers
// instance. Its fields are intended to be set/updated by TokenomicsModuleKeepersOptFn
// functions which are passed during integration construction.
type tokenomicsModuleKeepersConfig struct {
	tokenLogicModules []tlm.TokenLogicModule
	initKeepersFns    []func(context.Context, *TokenomicsModuleKeepers, *stakingkeeper.Keeper) context.Context
	// moduleParams is a map of module names to their respective module parameters.
	// This is used to set the initial module parameters in the keeper.
	moduleParams map[string]cosmostypes.Msg

	// proposerConsAddr and proposerValOperatorAddr are used to configure the block proposer
	proposerConsAddr        cosmostypes.ConsAddress
	proposerValOperatorAddr cosmostypes.ValAddress
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
		OwnerAddress:         sample.AccAddressBech32(),
	}

	// Initialize the in-memory database.
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	// Initialize the codec and other necessary components.
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// The onchain governance address.
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the test application.
	application := apptypes.Application{
		Address:        sample.AccAddressBech32(),
		Stake:          &cosmostypes.Coin{Denom: "upokt", Amount: cosmosmath.NewInt(100000)},
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}

	supplierOwnerAddr := sample.AccAddressBech32()
	proposerConsAddr := sample.ConsAddress()
	proposerValOperatorAddr := sample.ValOperatorAddressBech32()

	// The list of services that the supplier is staking for.
	services := []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: service.Id,
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{
					Address:            supplierOwnerAddr,
					RevSharePercentage: uint64(100),
				},
			},
		},
	}
	// Prepare the test supplier.
	supplier := sharedtypes.Supplier{
		OwnerAddress:    supplierOwnerAddr,
		OperatorAddress: supplierOwnerAddr,
		Stake:           &cosmostypes.Coin{Denom: "upokt", Amount: cosmosmath.NewInt(100000)},
		Services:        services,
	}

	sdkCtx := cosmostypes.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	sdkCtx = sdkCtx.WithProposer(proposerConsAddr)

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

	mockApplicationKeeper.EXPECT().
		GetParams(gomock.Any()).
		Return(apptypes.Params{}).
		AnyTimes()

	// Mock the supplier keeper.
	mockSupplierKeeper := mocks.NewMockSupplierKeeper(ctrl)
	// Mock SetAndIndexDehydratedSupplier.
	mockSupplierKeeper.EXPECT().
		SetAndIndexDehydratedSupplier(gomock.Any(), gomock.Any()).
		AnyTimes()
	mockSupplierKeeper.EXPECT().
		SetDehydratedSupplier(gomock.Any(), gomock.Any()).
		AnyTimes()

	// Get test supplier if the address matches.
	mockSupplierKeeper.EXPECT().
		GetSupplier(gomock.Any(), gomock.Eq(supplier.OperatorAddress)).
		Return(supplier, true).
		AnyTimes()
	mockSupplierKeeper.EXPECT().
		GetDehydratedSupplier(gomock.Any(), gomock.Eq(supplier.OperatorAddress)).
		Return(supplier, true).
		AnyTimes()
	mockSupplierKeeper.EXPECT().
		GetDehydratedSupplier(gomock.Any(), gomock.Not(supplier.OperatorAddress)).
		Return(sharedtypes.Supplier{}, false).
		AnyTimes()
	mockSupplierKeeper.EXPECT().
		GetSupplierActiveServiceConfig(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, supplierInstance *sharedtypes.Supplier, serviceId string) []*sharedtypes.SupplierServiceConfig {
			if supplier.OperatorAddress != supplierInstance.OperatorAddress {
				return []*sharedtypes.SupplierServiceConfig{}
			}
			if serviceId != service.Id {
				return []*sharedtypes.SupplierServiceConfig{}
			}

			return services
		}).
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

	// Mock the staking keeper
	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)
	validator := stakingtypes.Validator{
		OperatorAddress: proposerValOperatorAddr,
	}
	mockStakingKeeper.EXPECT().
		GetValidatorByConsAddr(gomock.Any(), proposerConsAddr).
		Return(validator, nil).
		AnyTimes()
	mockStakingKeeper.EXPECT().
		GetValidatorByConsAddr(gomock.Any(), gomock.Any()).
		Return(stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound).
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

	targetNumRelays := servicetypes.DefaultTargetNumRelays
	relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(
		sdkCtx,
		log.NewNopLogger(),
		service.Id,
		targetNumRelays,
		targetNumRelays,
	)
	mockServiceKeeper.EXPECT().
		GetRelayMiningDifficulty(gomock.Any(), gomock.Any()).
		Return(relayMiningDifficulty, true).
		AnyTimes()

	tokenLogicModules := tlm.NewDefaultTokenLogicModules()

	// Create mock staking and distribution keepers
	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)
	mockDistributionKeeper := mocks.NewMockDistributionKeeper(ctrl)

	// Set up mock expectations for staking keeper
	// Create a sample validator for testing
	sampleValidator := stakingtypes.Validator{
		OperatorAddress: sample.AccAddress(),
		Status:          stakingtypes.Bonded,
	}

	// Mock GetValidatorByConsAddr to return the sample validator
	mockStakingKeeper.EXPECT().
		GetValidatorByConsAddr(gomock.Any(), gomock.Any()).
		Return(sampleValidator, nil).
		AnyTimes()

	// Mock AllocateTokensToValidator to succeed
	mockDistributionKeeper.EXPECT().
		AllocateTokensToValidator(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
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
		mockStakingKeeper,
		mockDistributionKeeper,
		tokenLogicModules,
	)

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
		migrationtypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
	)

	// Construct a multistore & mount store keys for each keeper that will interact with the state store.
	stateStore := integration.CreateMultiStore(keys, log.NewNopLogger())

	// Use the test logger by default (i.e. if none is given).
	if logger == nil {
		logger = log.NewTestLogger(t)
	}

	// Add a block proposer address to the context
	var proposerConsAddr cosmostypes.ConsAddress
	var proposerValOperatorAddr cosmostypes.ValAddress
	if cfg.proposerConsAddr != nil && cfg.proposerValOperatorAddr != nil {
		proposerConsAddr = cfg.proposerConsAddr
		proposerValOperatorAddr = cfg.proposerValOperatorAddr
	} else {
		proposerConsAddr = sample.ConsAddress()
		proposerValOperatorAddr = sample.ValOperatorAddress()
	}

	// Prepare the context
	ctx = cosmostypes.NewContext(stateStore, cmtproto.Header{}, false, logger)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	// Set the proposer address in the context so that TLMs can resolve validators
	sdkCtx = sdkCtx.WithProposer(proposerConsAddr)
	ctx = sdkCtx

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
			minttypes.ModuleName:           {authtypes.Minter},
			suppliertypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
			apptypes.ModuleName:            {authtypes.Minter, authtypes.Burner},
			tokenomicstypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
			distrtypes.ModuleName:          nil,
			stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
			stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
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

	// Construct a real staking keeper for validator/delegator operations
	// Use the project's validator address prefix (pokt + valoper = poktvaloper)
	valAddrCodec := addresscodec.NewBech32Codec(app.AccountAddressPrefix + "valoper")
	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		authority.String(),
		addrCodec,
		valAddrCodec,
	)
	require.NoError(t, stakingKeeper.SetParams(sdkCtx, stakingtypes.DefaultParams()))

	// We will pass the concrete stakingKeeper to the option functions

	// Construct a real distribution keeper for reward distribution
	distKeeper := distrkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distrtypes.ModuleName,
		authority.String(),
	)
	// Note: Distribution keeper doesn't have SetParams method in newer versions

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
		stakingKeeper,
		distKeeper,
		cfg.tokenLogicModules,
	)

	require.NoError(t, tokenomicsKeeper.SetParams(sdkCtx, tokenomicstypes.DefaultParams()))

	if params, ok := cfg.moduleParams[tokenomicstypes.ModuleName]; ok {
		err := tokenomicsKeeper.SetParams(ctx, *params.(*tokenomicstypes.Params))
		require.NoError(t, err)
	}

	migrationKeeper := migrationkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[migrationtypes.StoreKey]),
		logger,
		authority.String(),
		accountKeeper,
		bankKeeper,
		sharedKeeper,
		appKeeper,
		supplierKeeper,
	)

	require.NoError(t, migrationKeeper.SetParams(sdkCtx, migrationtypes.DefaultParams()))

	if params, ok := cfg.moduleParams[migrationtypes.ModuleName]; ok {
		err := migrationKeeper.SetParams(ctx, *params.(*migrationtypes.Params))
		require.NoError(t, err)
	}

	keepers := TokenomicsModuleKeepers{
		Keeper:             &tokenomicsKeeper,
		AccountKeeper:      &accountKeeper,
		BankKeeper:         &bankKeeper,
		ApplicationKeeper:  &appKeeper,
		SupplierKeeper:     &supplierKeeper,
		ProofKeeper:        &proofKeeper,
		SharedKeeper:       &sharedKeeper,
		SessionKeeper:      &sessionKeeper,
		ServiceKeeper:      &serviceKeeper,
		StakingKeeper:      &stakingKeeper,
		DistributionKeeper: &distKeeper,
		MigrationKeeper:    &migrationKeeper,

		Codec: cdc,
	}

	// Apply any options to update the keepers or context prior to returning them.
	ctx = sdkCtx
	for _, fn := range cfg.initKeepersFns {
		ctx = fn(ctx, &keepers, &stakingKeeper)
	}

	return keepers, ctx
}

// WithService is an option to set the service in the tokenomics module keepers.
func WithService(service sharedtypes.Service) TokenomicsModuleKeepersOptFn {
	setService := func(ctx context.Context, keepers *TokenomicsModuleKeepers, _ *stakingkeeper.Keeper) context.Context {
		keepers.SetService(ctx, service)
		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setService)
	}
}

// WithApplication is an option to set the application in the tokenomics module keepers.
func WithApplication(applicaion apptypes.Application) TokenomicsModuleKeepersOptFn {
	setApp := func(ctx context.Context, keepers *TokenomicsModuleKeepers, _ *stakingkeeper.Keeper) context.Context {
		keepers.SetApplication(ctx, applicaion)
		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setApp)
	}
}

// WithSupplier is an option to set the supplier in the tokenomics module keepers.
func WithSupplier(supplier sharedtypes.Supplier) TokenomicsModuleKeepersOptFn {
	setSupplier := func(ctx context.Context, keepers *TokenomicsModuleKeepers, _ *stakingkeeper.Keeper) context.Context {
		keepers.SetAndIndexDehydratedSupplier(ctx, supplier)
		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setSupplier)
	}
}

// WithProposerAddr is an option to set the proposer address in the context used
// by the tokenomics module keepers.
func WithProposerAddr(addr string) TokenomicsModuleKeepersOptFn {
	setProposerAddrAndValidator := func(ctx context.Context, keepers *TokenomicsModuleKeepers, stakingKeeper *stakingkeeper.Keeper) context.Context {
		consAddr, err := cosmostypes.ConsAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		sdkCtx = sdkCtx.WithProposer(consAddr)

		// Create a validator and update the context with the corresponding consensus address
		// This ensures the validator and proposer address are properly matched
		actualConsAddr, err := createValidatorForProposer(sdkCtx, stakingKeeper)
		if err != nil {
			panic(fmt.Sprintf("failed to create validator for proposer: %v", err))
		}

		// Update the context to use the consensus address that matches our validator
		sdkCtx = sdkCtx.WithProposer(actualConsAddr)

		return sdkCtx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setProposerAddrAndValidator)
	}
}

// WithBlockProposer is an option to set the proposer address in the context used
// by the tokenomics module keepers and configures the staking keeper mock to return
// the correct validator for the consensus address.
func WithBlockProposer(
	consAddr cosmostypes.ConsAddress,
	valOperatorAddr cosmostypes.ValAddress,
) TokenomicsModuleKeepersOptFn {
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.proposerConsAddr = consAddr
		cfg.proposerValOperatorAddr = valOperatorAddr

		// Set the proposer address in the context
		setProposerAddr := func(ctx context.Context, keepers *TokenomicsModuleKeepers, stakingKeeper *stakingkeeper.Keeper) context.Context {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			sdkCtx = sdkCtx.WithProposer(consAddr)
			return sdkCtx
		}
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

// WithModuleParams returns a KeeperOptionFn that sets the moduleParams field
// on the keeperConfig.
func WithModuleParams(moduleParams map[string]cosmostypes.Msg) TokenomicsModuleKeepersOptFn {
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.moduleParams = moduleParams
	}
}

// createValidatorForProposer creates a validator for integration tests and returns the consensus address
// that matches the validator's public key. This ensures proper validator reward distribution.
func createValidatorForProposer(ctx context.Context, stakingKeeper *stakingkeeper.Keeper) (cosmostypes.ConsAddress, error) {
	// Create a consensus private/public key pair for testing
	consPrivKey := ed25519.GenPrivKey()
	consPubKey := consPrivKey.PubKey()

	// Get the consensus address from the public key
	consAddr := cosmostypes.ConsAddress(consPubKey.Address())

	// Use standard account address format for validator operator address
	// This matches the pattern used in the mock setup where sample.AccAddress() is used
	operatorAddress := sample.AccAddress()

	// Convert consensus public key to Any for storage in validator
	consPubKeyAny, err := codectypes.NewAnyWithValue(consPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert consensus pubkey: %v", err)
	}

	validator := stakingtypes.Validator{
		OperatorAddress: operatorAddress,
		ConsensusPubkey: consPubKeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          cosmosmath.NewInt(1000000), // 1M tokens staked
		DelegatorShares: cosmosmath.LegacyNewDec(1000000),
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          cosmosmath.LegacyNewDecWithPrec(5, 2),  // 5% commission
				MaxRate:       cosmosmath.LegacyNewDecWithPrec(20, 2), // 20% max
				MaxChangeRate: cosmosmath.LegacyNewDecWithPrec(1, 2),  // 1% max change
			},
			UpdateTime: cosmostypes.UnwrapSDKContext(ctx).BlockTime(),
		},
	}

	// Use the concrete staking keeper to set the validator
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Set the validator in state
	if err := stakingKeeper.SetValidator(sdkCtx, validator); err != nil {
		return nil, fmt.Errorf("failed to set validator: %v", err)
	}

	// SetValidatorByConsAddr creates the mapping from consensus address to validator
	if err := stakingKeeper.SetValidatorByConsAddr(sdkCtx, validator); err != nil {
		return nil, fmt.Errorf("failed to set validator by consensus address: %v", err)
	}

	return consAddr, nil
}

// WithProofRequirement is an option to enable or disable the proof requirement
// in the tokenomics module keepers by setting the proof request probability to
// 1 or 0, respectively whie setting the proof requirement threshold to 0 or
// MaxInt64, respectively.
func WithProofRequirement(proofRequired bool) TokenomicsModuleKeepersOptFn {
	setProofRequirement := func(ctx context.Context, keepers *TokenomicsModuleKeepers, _ *stakingkeeper.Keeper) context.Context {
		proofParams := keepers.ProofKeeper.GetParams(ctx)
		if proofRequired {
			// Require a proof 100% of the time probabilistically speaking.
			proofParams.ProofRequestProbability = 1
			// Require a proof of any claim amount (i.e. anything greater than 0).
			proofRequirementThreshold := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)
			proofParams.ProofRequirementThreshold = &proofRequirementThreshold
		} else {
			// Never require a proof probabilistically speaking.
			proofParams.ProofRequestProbability = 0
			// Require a proof for MaxInt64 claim amount (i.e. should never trigger).
			proofRequirementThreshold := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, math.MaxInt64)
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

// WithDefaultModuleBalances mints an arbitrary amount of uPOKT to the respective modules.
func WithDefaultModuleBalances() func(cfg *tokenomicsModuleKeepersConfig) {
	return WithModuleAccountBalances(map[string]int64{
		apptypes.ModuleName:      1000000000000,
		suppliertypes.ModuleName: 1000000000000,
	})
}

// WithModuleAccountBalances mints the given amount of uPOKT to the respective modules.
func WithModuleAccountBalances(moduleAccountBalances map[string]int64) func(cfg *tokenomicsModuleKeepersConfig) {
	setModuleAccountBalances := func(ctx context.Context, keepers *TokenomicsModuleKeepers, _ *stakingkeeper.Keeper) context.Context {
		for moduleName, balanceCoin := range moduleAccountBalances {
			err := keepers.MintCoins(ctx, moduleName, cosmostypes.NewCoins(cosmostypes.NewInt64Coin(pocket.DenomuPOKT, balanceCoin)))
			if err != nil {
				panic(err)
			}
		}

		return ctx
	}
	return func(cfg *tokenomicsModuleKeepersConfig) {
		cfg.initKeepersFns = append(cfg.initKeepersFns, setModuleAccountBalances)
	}
}
