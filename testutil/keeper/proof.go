package keeper

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	cmtabcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	applicationmocks "github.com/pokt-network/poktroll/testutil/application/mocks"
	"github.com/pokt-network/poktroll/testutil/proof/mocks"
	sessionmocks "github.com/pokt-network/poktroll/testutil/session/mocks"
	suppliermocks "github.com/pokt-network/poktroll/testutil/supplier/mocks"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	application "github.com/pokt-network/poktroll/x/application/module"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	proof "github.com/pokt-network/poktroll/x/proof/module"
	"github.com/pokt-network/poktroll/x/proof/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	session "github.com/pokt-network/poktroll/x/session/module"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	supplier "github.com/pokt-network/poktroll/x/supplier/module"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_IN_THIS_COMMIT: comment...
type (
	BankKeeper     = bankkeeper.Keeper
	AuthzKeeper    = authzkeeper.Keeper
	GovKeeper      = govkeeper.Keeper
	SessionKeeper  = sessionkeeper.Keeper
	SupplierKeeper = supplierkeeper.Keeper
	AppKeeper      = appkeeper.Keeper
)

// Collect store keys for all keepers which be constructed & interact with the state store.
var storeKeys = storetypes.NewKVStoreKeys(
	types.StoreKey,
	sessiontypes.StoreKey,
	suppliertypes.StoreKey,
	apptypes.StoreKey,
	gatewaytypes.StoreKey,
	authtypes.StoreKey,
	banktypes.StoreKey,
	authzkeeper.StoreKey,
	govtypes.StoreKey,
)

// TODO_IN_THIS_COMMIT: move & update comment.
// App is a test application that can be used to test the integration of modules.
type IntegrationApp struct {
	*baseapp.BaseApp

	// TODO_IN_THIS_COMMIT: export?
	ctx           sdk.Context
	logger        log.Logger
	moduleManager module.Manager
	queryHelper   *baseapp.QueryServiceTestHelper
}

// ProofModuleKeepers is an aggregation of the proof keeper and all its dependency
// keepers, and the codec that they share. Each keeper is embedded such that the
// ProofModuleKeepers implements all the interfaces of the keepers.
// To call a method which is common to multiple keepers (e.g. `#SetParams()`),
// the field corresponding to the desired keeper on which to call the method
// MUST be specified (e.g. `keepers.AccountKeeper#SetParams()`).
type ProofModuleKeepers struct {
	keeper.Keeper
	//SessionKeeper sessionkeeper.Keeper
	//prooftypes.SessionKeeper
	SessionKeeper
	//prooftypes.SupplierKeeper
	SupplierKeeper
	//prooftypes.ApplicationKeeper
	AppKeeper
	//AccountKeeper
	authkeeper.AccountKeeper
	BankKeeper
	AuthzKeeper
	GovKeeper

	Codec             *codec.ProtoCodec
	InterfaceRegistry codectypes.InterfaceRegistry
	BaseApp           *baseapp.BaseApp
}

// ProofKeepersOpt is a function which receives and potentially modifies the context
// and proof keepers during construction of the aggregation.
type ProofKeepersOpt func(context.Context, *ProofModuleKeepers) context.Context

// ProofKeeper is a helper function to create a proof keeper and a context. It uses
// mocked dependencies only.
func ProofKeeper(t testing.TB) (keeper.Keeper, context.Context) {
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
	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockAppKeeper := mocks.NewMockApplicationKeeper(ctrl)
	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockSessionKeeper,
		mockAppKeeper,
		mockAccountKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	return k, ctx
}

// NewProofModuleKeepers is a helper function to create a proof keeper and a context. It uses
// real dependencies for all keepers except the bank keeper, which is mocked as it's not used
// directly by the proof keeper or its dependencies.
func NewProofModuleKeepers(t testing.TB, opts ...ProofKeepersOpt) (_ *ProofModuleKeepers, ctx context.Context) {
	t.Helper()

	// Construct a multistore & mount store keys for each keeper that will interact with the state store.
	stateStore := integration.CreateMultiStore(storeKeys, log.NewNopLogger())

	logger := log.NewTestLogger(t)
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, logger)

	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(
		codectypes.InterfaceRegistryOptions{
			ProtoFiles: proto.HybridResolver,
			SigningOptions: signing.Options{
				//FileResolver:          nil,
				//TypeResolver:          nil,
				AddressCodec:          addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
				ValidatorAddressCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
				//CustomGetSigners:      nil,
				//MaxRecursionDepth:     0,
			},
		},
	)
	require.NoError(t, err)

	authtypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)

	cdc := codec.NewProtoCodec(interfaceRegistry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Mock the bank keeper.
	ctrl := gomock.NewController(t)

	// Construct a real account keeper so that public keys can be queried.
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		map[string][]string{minttypes.ModuleName: {authtypes.Minter}},
		addresscodec.NewBech32Codec(app.AccountAddressPrefix),
		app.AccountAddressPrefix,
		authority.String(),
	)

	baseApp := NewIntegrationBaseApp(logger, storeKeys, interfaceRegistry)

	authzKeeper := authzkeeper.NewKeeper(
		runtime.NewKVStoreService(storeKeys[authzkeeper.StoreKey]),
		cdc,
		baseApp.MsgServiceRouter(),
		accountKeeper,
	)

	// Construct bank keeper.
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[banktypes.StoreKey]),
		accountKeeper,
		nil,
		authority.String(),
		logger,
	)

	// TODO: add governance (and dependency) module(s) to support more scenarios:
	//
	// TODO: need to set "bonded_tokens_pool" module account in account keeper.
	//stakingKeeper := stakingkeeper.NewKeeper(
	//	cdc,
	//	runtime.NewKVStoreService(storeKeys[stakingtypes.StoreKey]),
	//	accountKeeper,
	//	bankKeeper,
	//	authority.String(),
	//	addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
	//	addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	//)
	//
	//distKeeper := distrkeeper.NewKeeper(
	//	cdc,
	//	runtime.NewKVStoreService(storeKeys[distrtypes.StoreKey]),
	//	accountKeeper,
	//	bankKeeper,
	//	stakingKeeper,
	//	authority.String(), // fee collector
	//	authority.String(), // authority
	//)
	//
	//// Construct governance keeper.
	//govKeeper := govkeeper.NewKeeper(
	//	cdc,
	//	runtime.NewKVStoreService(storeKeys[govtypes.StoreKey]),
	//	accountKeeper,
	//	bankKeeper,
	//	stakingKeeper,
	//	distKeeper,
	//	baseApp.MsgServiceRouter(),
	//	govtypes.Config{},
	//	authority.String(),
	//)

	// Construct gateway keeper.
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[gatewaytypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
	)
	require.NoError(t, gatewayKeeper.SetParams(ctx, gatewaytypes.DefaultParams()))

	// Construct an application keeper to add apps to sessions.
	appKeeper := appkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
		applicationmocks.NewMockBankKeeper(ctrl),
		accountKeeper,
		gatewayKeeper,
	)
	require.NoError(t, appKeeper.SetParams(ctx, apptypes.DefaultParams()))

	// Construct a real supplier keeper to add suppliers to sessions.
	supplierKeeper := supplierkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[suppliertypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		suppliermocks.NewMockBankKeeper(ctrl),
	)
	require.NoError(t, supplierKeeper.SetParams(ctx, suppliertypes.DefaultParams()))

	// Construct a real session keeper so that sessions can be queried.
	sessionKeeper := sessionkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[sessiontypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		sessionmocks.NewMockBankKeeper(ctrl),
		appKeeper,
		supplierKeeper,
	)
	require.NoError(t, sessionKeeper.SetParams(ctx, sessiontypes.DefaultParams()))

	// Construct a real proof keeper so that claims & proofs can be created.
	proofKeeper := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[types.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		sessionKeeper,
		appKeeper,
		accountKeeper,
	)
	require.NoError(t, proofKeeper.SetParams(ctx, types.DefaultParams()))

	keepers := &ProofModuleKeepers{
		Keeper:         proofKeeper,
		SessionKeeper:  sessionKeeper,
		SupplierKeeper: supplierKeeper,
		AppKeeper:      appKeeper,
		AccountKeeper:  accountKeeper,
		BankKeeper:     bankKeeper,
		AuthzKeeper:    authzKeeper,
		//GovKeeper:      *govKeeper,

		Codec:             cdc,
		BaseApp:           baseApp,
		InterfaceRegistry: interfaceRegistry,
	}

	// Apply any options to update the keepers or context prior to returning them.
	for _, opt := range opts {
		ctx = opt(ctx, keepers)
	}

	return keepers, ctx
}

// AddServiceActors adds a supplier and an application for a specific
// service so a successful session can be generated for testing purposes.
func (keepers *ProofModuleKeepers) AddServiceActors(
	ctx context.Context,
	t *testing.T,
	service *sharedtypes.Service,
	supplierAddr string,
	appAddr string,
) {
	t.Helper()

	keepers.SetSupplier(ctx, sharedtypes.Supplier{
		Address: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	keepers.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: service},
		},
	})
}

// GetSessionHeader is a helper to retrieve the session header
// for a specific (app, service, height).
func (keepers *ProofModuleKeepers) GetSessionHeader(
	ctx context.Context,
	t *testing.T,
	appAddr string,
	service *sharedtypes.Service,
	blockHeight int64,
) *sessiontypes.SessionHeader {
	t.Helper()

	sessionRes, err := keepers.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			Service:            service,
			BlockHeight:        blockHeight,
		},
	)
	require.NoError(t, err)

	return sessionRes.GetSession().GetHeader()
}

// TODO_IN_THIS_COMMIT: move up..
type (
	AuthModule     = auth.AppModule
	BankModule     = bank.AppModule
	AuthzModule    = authzmodule.AppModule
	GovModule      = gov.AppModule
	SessionModule  = session.AppModule
	SupplierModule = supplier.AppModule
	AppModule      = application.AppModule
	ProofModule    = proof.AppModule
)

// TODO_IN_THIS_COMMIT: move up..
type ProofModules struct {
	ProofModuleKeepers

	*AuthModule
	*BankModule
	*AuthzModule
	*GovModule
	*SessionModule
	*SupplierModule
	*AppModule
	*ProofModule
}

func (pmk *ProofModuleKeepers) NewModules() *ProofModules {
	// Construct modules necessary for test scenario:
	authModule := auth.NewAppModule(pmk.Codec, pmk.AccountKeeper, authsims.RandomGenesisAccounts, nil)
	bankModule := bank.NewAppModule(pmk.Codec, pmk.BankKeeper, pmk.AccountKeeper, nil)
	authzModule := authzmodule.NewAppModule(pmk.Codec, pmk.AuthzKeeper, pmk.AccountKeeper, pmk.BankKeeper, pmk.InterfaceRegistry)
	//govModule := gov.NewAppModule(pmk.Codec, &pmk.GovKeeper, pmk.AccountKeeper, pmk.BankKeeper, nil)
	sessionModule := session.NewAppModule(pmk.Codec, pmk.SessionKeeper, pmk.AccountKeeper, pmk.BankKeeper)
	supplierModule := supplier.NewAppModule(pmk.Codec, pmk.SupplierKeeper, pmk.AccountKeeper, pmk.BankKeeper)
	appModule := application.NewAppModule(pmk.Codec, pmk.AppKeeper, pmk.AccountKeeper, pmk.BankKeeper)
	proofModule := proof.NewAppModule(pmk.Codec, pmk.Keeper, pmk.AccountKeeper)

	return &ProofModules{
		// NB: copying the keepers (by value).
		ProofModuleKeepers: *pmk,
		AuthModule:         &authModule,
		BankModule:         &bankModule,
		AuthzModule:        &authzModule,
		//GovModule:          &govModule,
		SessionModule:  &sessionModule,
		SupplierModule: &supplierModule,
		AppModule:      &appModule,
		ProofModule:    &proofModule,
	}
}

func (pm *ProofModules) NewIntegrationApp(ctx context.Context) *IntegrationApp {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Build app from modules
	integrationApp := NewIntegrationAppWithBaseApp(
		sdkCtx,
		sdkCtx.Logger(),
		storeKeys,
		pm.Codec,
		map[string]appmodule.AppModule{
			authtypes.ModuleName: pm.AuthModule,
			banktypes.ModuleName: pm.BankModule,
			authz.ModuleName:     pm.AuthzModule,
			// stakingtypes.ModuleName:  pm.StakingModule,
			// distrtypes.ModuleName:    pm.DistModule,
			// govtypes.ModuleName:      pm.GovModule,
			sessiontypes.ModuleName:  pm.SessionModule,
			suppliertypes.ModuleName: pm.SupplierModule,
			apptypes.ModuleName:      pm.AppModule,
			prooftypes.ModuleName:    pm.ProofModule,
		},
		pm.BaseApp,
		pm.InterfaceRegistry,
	)

	// Register message & query server implementations
	authtypes.RegisterMsgServer(
		integrationApp.MsgServiceRouter(),
		authkeeper.NewMsgServerImpl(pm.AccountKeeper),
	)
	authtypes.RegisterQueryServer(
		integrationApp.QueryHelper(),
		authkeeper.NewQueryServer(pm.AccountKeeper),
	)

	sessiontypes.RegisterMsgServer(
		integrationApp.MsgServiceRouter(),
		sessionkeeper.NewMsgServerImpl(pm.SessionKeeper),
	)
	// TODO_IN_THIS_COMMIT: register query server?

	suppliertypes.RegisterMsgServer(
		integrationApp.MsgServiceRouter(),
		supplierkeeper.NewMsgServerImpl(pm.SupplierKeeper),
	)
	// TODO_IN_THIS_COMMIT: register query server?

	apptypes.RegisterMsgServer(
		integrationApp.MsgServiceRouter(),
		appkeeper.NewMsgServerImpl(pm.AppKeeper),
	)
	// TODO_IN_THIS_COMMIT: register query server?

	prooftypes.RegisterMsgServer(
		integrationApp.MsgServiceRouter(),
		keeper.NewMsgServerImpl(pm.Keeper),
	)
	// TODO_IN_THIS_COMMIT: register query server?

	authz.RegisterMsgServer(
		integrationApp.MsgServiceRouter(),
		pm.AuthzKeeper,
	)

	return integrationApp
}

// TODO_IN_THIS_COMMIT: move & update comment.
//
// RunMsg provides the ability to run a message and return the response.
// In order to run a message, the application must have a handler for it.
// These handlers are registered on the application message service router.
// The result of the message execution is returned as an Any type.
// That any type can be unmarshaled to the expected response type.
// If the message execution fails, an error is returned.
func (app *IntegrationApp) RunMsg(msg sdk.Msg, option ...integration.Option) (*codectypes.Any, error) {
	// set options
	cfg := &integration.Config{}
	for _, opt := range option {
		opt(cfg)
	}

	if cfg.AutomaticCommit {
		defer app.Commit()
	}

	if cfg.AutomaticFinalizeBlock {
		height := app.LastBlockHeight() + 1
		if _, err := app.FinalizeBlock(&cmtabcitypes.RequestFinalizeBlock{Height: height}); err != nil {
			return nil, fmt.Errorf("failed to run finalize block: %w", err)
		}
	}

	app.logger.Info("Running msg", "msg", msg.String())

	handler := app.MsgServiceRouter().Handler(msg)
	if handler == nil {
		return nil, fmt.Errorf("handler is nil, can't route message %s: %+v", sdk.MsgTypeURL(msg), msg)
	}

	msgResult, err := handler(app.ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute message %s: %w", sdk.MsgTypeURL(msg), err)
	}

	var response *codectypes.Any
	if len(msgResult.MsgResponses) > 0 {
		msgResponse := msgResult.MsgResponses[0]
		if msgResponse == nil {
			return nil, fmt.Errorf("got nil msg response %s in message result: %s", sdk.MsgTypeURL(msg), msgResult.String())
		}

		response = msgResponse
	}

	return response, nil
}

// TODO_IN_THIS_COMMIT: move and udpate comment
//
// QueryHelper returns the application query helper.
// It can be used when registering query services.
func (app *IntegrationApp) QueryHelper() *baseapp.QueryServiceTestHelper {
	return app.queryHelper
}

// CreateMultiStore is a helper for setting up multiple stores for provided modules.
func CreateMultiStore(keys map[string]*storetypes.KVStoreKey, logger log.Logger) storetypes.CommitMultiStore {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())

	for key := range keys {
		cms.MountStoreWithDB(keys[key], storetypes.StoreTypeIAVL, db)
	}

	_ = cms.LoadLatestVersion()
	return cms
}

// TODO_IN_THIS_COMMIT: move to top.
const appName = "integration-app"

func NewIntegrationAppWithBaseApp(
	sdkCtx sdk.Context,
	logger log.Logger,
	keys map[string]*storetypes.KVStoreKey,
	appCodec codec.Codec,
	modules map[string]appmodule.AppModule,
	baseApp *baseapp.BaseApp,
	interfaceRegistry codectypes.InterfaceRegistry,
) *IntegrationApp {
	moduleManager := module.NewManagerFromMap(modules)
	basicModuleManager := module.NewBasicManagerFromManager(moduleManager, nil)
	basicModuleManager.RegisterInterfaces(interfaceRegistry)

	baseApp.SetInitChainer(func(ctx sdk.Context, _ *cmtabcitypes.RequestInitChain) (*cmtabcitypes.ResponseInitChain, error) {
		for _, mod := range modules {
			if m, ok := mod.(module.HasGenesis); ok {
				m.InitGenesis(ctx, appCodec, m.DefaultGenesis(appCodec))
			}
		}

		return &cmtabcitypes.ResponseInitChain{}, nil
	})

	baseApp.SetBeginBlocker(func(_ sdk.Context) (sdk.BeginBlock, error) {
		return moduleManager.BeginBlock(sdkCtx)
	})
	baseApp.SetEndBlocker(func(_ sdk.Context) (sdk.EndBlock, error) {
		return moduleManager.EndBlock(sdkCtx)
	})

	if keys[consensusparamtypes.StoreKey] != nil {
		// set baseApp param store
		consensusParamsKeeper := consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), authtypes.NewModuleAddress("gov").String(), runtime.EventService{})
		baseApp.SetParamStore(consensusParamsKeeper.ParamsStore)

		if err := baseApp.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("failed to load application version from store: %w", err))
		}

		if _, err := baseApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName, ConsensusParams: simtestutil.DefaultConsensusParams}); err != nil {
			panic(fmt.Errorf("failed to initialize application: %w", err))
		}
	} else {
		if err := baseApp.LoadLatestVersion(); err != nil {
			panic(fmt.Errorf("failed to load application version from store: %w", err))
		}

		if _, err := baseApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName}); err != nil {
			panic(fmt.Errorf("failed to initialize application: %w", err))
		}
	}

	baseApp.Commit()

	ctx := sdkCtx.WithBlockHeader(cmtproto.Header{ChainID: appName}).WithIsCheckTx(true)

	return &IntegrationApp{
		BaseApp:       baseApp,
		logger:        logger,
		ctx:           ctx,
		moduleManager: *moduleManager,
		queryHelper:   baseapp.NewQueryServerTestHelper(ctx, interfaceRegistry),
	}
}

func NewIntegrationBaseApp(
	logger log.Logger,
	keys map[string]*storetypes.KVStoreKey,
	interfaceRegistry codectypes.InterfaceRegistry,
) *baseapp.BaseApp {
	db := dbm.NewMemDB()

	txConfig := tx.NewTxConfig(codec.NewProtoCodec(interfaceRegistry), tx.DefaultSignModes)

	baseApp := baseapp.NewBaseApp(appName, logger, db, txConfig.TxDecoder(), baseapp.SetChainID(appName))
	baseApp.MountKVStores(keys)

	router := baseapp.NewMsgServiceRouter()
	router.SetInterfaceRegistry(interfaceRegistry)
	baseApp.SetMsgServiceRouter(router)

	return baseApp
}

// WithBlockHash sets the initial block hash for the context and returns the updated context.
func WithBlockHash(hash []byte) ProofKeepersOpt {
	return func(ctx context.Context, _ *ProofModuleKeepers) context.Context {
		return SetBlockHash(ctx, hash)
	}
}

// SetBlockHash updates the block hash for the given context and returns the updated context.
func SetBlockHash(ctx context.Context, hash []byte) context.Context {
	return sdk.UnwrapSDKContext(ctx).WithHeaderHash(hash)
}

// WithBlockHeight sets the initial block height for the context and returns the updated context.
func WithBlockHeight(height int64) ProofKeepersOpt {
	return func(ctx context.Context, _ *ProofModuleKeepers) context.Context {
		return SetBlockHeight(ctx, height)
	}
}

// SetBlockHeight updates the block height for the given context and returns the updated context.
func SetBlockHeight(ctx context.Context, height int64) context.Context {
	return sdk.UnwrapSDKContext(ctx).WithBlockHeight(height)
}
