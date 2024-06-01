package integration

import (
	"testing"
	"time"

	"cosmossdk.io/core/appmodule"
	coreheader "cosmossdk.io/core/header"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtabcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	application "github.com/pokt-network/poktroll/x/application/module"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gateway "github.com/pokt-network/poktroll/x/gateway/module"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	proof "github.com/pokt-network/poktroll/x/proof/module"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	session "github.com/pokt-network/poktroll/x/session/module"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	shared "github.com/pokt-network/poktroll/x/shared/module"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomics "github.com/pokt-network/poktroll/x/tokenomics/module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const appName = "poktroll-integration-app"

// App is a test application that can be used to test the integration of modules.
type App struct {
	*baseapp.BaseApp

	Ctx           sdk.Context
	Cdc           codec.Codec
	Logger        log.Logger
	Authority     sdk.AccAddress
	ModuleManager module.Manager
	QueryHelper   *baseapp.QueryServiceTestHelper
}

func NewIntegrationApp(
	t *testing.T,
	sdkCtx sdk.Context,
	logger log.Logger,
	authority sdk.AccAddress,
	keys map[string]*storetypes.KVStoreKey,
	cdc codec.Codec,
	modules map[string]appmodule.AppModule,
	msgRouter *baseapp.MsgServiceRouter,
	// grpcRouter *baseapp.GRPCQueryRouter,
	queryHelper *baseapp.QueryServiceTestHelper,
) *App {
	t.Helper()

	db := dbm.NewMemDB()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	moduleManager := module.NewManagerFromMap(modules)
	basicModuleManager := module.NewBasicManagerFromManager(moduleManager, nil)
	basicModuleManager.RegisterInterfaces(interfaceRegistry)

	txConfig := authtx.NewTxConfig(codec.NewProtoCodec(interfaceRegistry), authtx.DefaultSignModes)
	bApp := baseapp.NewBaseApp(appName, logger, db, txConfig.TxDecoder(), baseapp.SetChainID(appName))
	bApp.MountKVStores(keys)

	bApp.SetInitChainer(
		func(ctx sdk.Context, _ *cmtabcitypes.RequestInitChain) (*cmtabcitypes.ResponseInitChain, error) {
			for _, mod := range modules {
				if m, ok := mod.(module.HasGenesis); ok {
					m.InitGenesis(ctx, cdc, m.DefaultGenesis(cdc))
				}
			}

			return &cmtabcitypes.ResponseInitChain{}, nil
		})

	bApp.SetBeginBlocker(func(_ sdk.Context) (sdk.BeginBlock, error) {
		return moduleManager.BeginBlock(sdkCtx)
	})
	bApp.SetEndBlocker(func(_ sdk.Context) (sdk.EndBlock, error) {
		return moduleManager.EndBlock(sdkCtx)
	})

	msgRouter.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetMsgServiceRouter(msgRouter)

	// grpcRouter.SetInterfaceRegistry(interfaceRegistry)

	err := bApp.LoadLatestVersion()
	require.NoError(t, err, "failed to load latest version")

	_, err = bApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName})
	require.NoError(t, err, "failed to initialize chain")

	_, err = bApp.Commit()
	require.NoError(t, err, "failed to commit")

	cometHeader := cmtproto.Header{
		ChainID: appName,
		Height:  2}
	ctx := sdkCtx.WithBlockHeader(cometHeader).WithIsCheckTx(true)

	return &App{
		BaseApp:       bApp,
		Logger:        logger,
		Authority:     authority,
		Ctx:           ctx,
		Cdc:           cdc,
		ModuleManager: *moduleManager,
		QueryHelper:   queryHelper,
	}
}

func NewCompleteIntegrationApp(t *testing.T) *App {
	t.Helper()

	// Register the codec for all the interfacesPrepare all the interfaces
	registry := codectypes.NewInterfaceRegistry()
	tokenomicstypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	gatewaytypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	sessiontypes.RegisterInterfaces(registry)
	apptypes.RegisterInterfaces(registry)
	suppliertypes.RegisterInterfaces(registry)
	prooftypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)

	// Prepare all the store keys
	storeKeys := storetypes.NewKVStoreKeys(
		tokenomicstypes.StoreKey,
		banktypes.StoreKey,
		gatewaytypes.StoreKey,
		sessiontypes.StoreKey,
		apptypes.StoreKey,
		suppliertypes.StoreKey,
		prooftypes.StoreKey,
		authtypes.StoreKey)

	// Prepare the context
	logger := log.NewNopLogger() // log.NewTestLogger(t)
	cms := CreateMultiStore(storeKeys, logger)
	ctx := sdk.NewContext(cms, cmtproto.Header{}, true, logger)

	// Get the authority address
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the account keeper
	addrCodec := addresscodec.NewBech32Codec(app.AccountAddressPrefix)
	macPerms := map[string][]string{
		banktypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		tokenomicstypes.ModuleName: {authtypes.Minter, authtypes.Burner},
		gatewaytypes.ModuleName:    {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		sessiontypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		apptypes.ModuleName:        {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		suppliertypes.ModuleName:   {authtypes.Minter, authtypes.Burner, authtypes.Staking},
	}

	// Prepare the account keeper and module
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		macPerms,
		addrCodec,
		app.AccountAddressPrefix,
		authority.String(),
	)
	authModule := auth.NewAppModule(
		cdc,
		accountKeeper,
		authsims.RandomGenesisAccounts,
		nil, // subspace is nil because we don't test params (which is legacy anyway)
	)

	blockedAddresses := map[string]bool{
		accountKeeper.GetAuthority(): false,
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[banktypes.StoreKey]),
		accountKeeper,
		blockedAddresses,
		authority.String(),
		logger)

	// Prepare the shared keeper and module
	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
	)
	sharedModule := shared.NewAppModule(
		cdc,
		sharedKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the gateway keeper and module
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
	)
	gatewayModule := gateway.NewAppModule(
		cdc,
		gatewayKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the application keeper and module
	applicationKeeper := appkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		gatewayKeeper,
		sharedKeeper,
	)
	applicationModule := application.NewAppModule(
		cdc,
		applicationKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the session keeper and module
	sessionKeeper := sessionkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[sessiontypes.StoreKey]),
		logger,
		authority.String(),
		accountKeeper,
		bankKeeper,
		applicationKeeper,
		nil, // supplierKeeper
		sharedKeeper,
	)
	sessionModule := session.NewAppModule(
		cdc,
		sessionKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the proof keeper and module
	proofKeeper := proofkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[prooftypes.StoreKey]),
		logger,
		authority.String(),
		sessionKeeper,
		applicationKeeper,
		accountKeeper,
		sharedKeeper,
	)
	proofModule := proof.NewAppModule(
		cdc,
		proofKeeper,
		accountKeeper,
	)

	// Prepare the tokenomics keeper and module
	tokenomicsKeeper := tokenomicskeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[tokenomicstypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		applicationKeeper,
		proofKeeper,
	)
	tokenomicsModule := tokenomics.NewAppModule(
		cdc,
		tokenomicsKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the message & query routers
	msgRouter := baseapp.NewMsgServiceRouter()
	queryRouter := baseapp.NewQueryServerTestHelper(ctx, registry)

	// Prepare the list of modules
	modules := map[string]appmodule.AppModule{
		tokenomicstypes.ModuleName: tokenomicsModule,
		sharedtypes.ModuleName:     sharedModule,
		gatewaytypes.ModuleName:    gatewayModule,
		apptypes.ModuleName:        applicationModule,
		prooftypes.ModuleName:      proofModule,
		authtypes.ModuleName:       authModule,
		sessiontypes.ModuleName:    sessionModule,
	}

	// Initialize the integration app
	integrationApp := NewIntegrationApp(
		t,
		ctx,
		logger,
		authority,
		storeKeys,
		cdc,
		modules,
		msgRouter,
		queryRouter,
	)

	// Register the message servers
	authtypes.RegisterMsgServer(msgRouter, authkeeper.NewMsgServerImpl(accountKeeper))
	tokenomicstypes.RegisterMsgServer(msgRouter, tokenomicskeeper.NewMsgServerImpl(tokenomicsKeeper))
	prooftypes.RegisterMsgServer(msgRouter, proofkeeper.NewMsgServerImpl(proofKeeper))

	// Register query servers
	tokenomicstypes.RegisterQueryServer(queryRouter, tokenomicsKeeper)
	prooftypes.RegisterQueryServer(queryRouter, proofKeeper)

	return integrationApp
}

// RunMsg provides the ability to run a message and return the response.
// In order to run a message, the application must have a handler for it.
// These handlers are registered on the application message service router.
// The result of the message execution is returned as an Any type.
// That any type can be unmarshaled to the expected response type.
// If the message execution fails, an error is returned.
func (app *App) RunMsg(t *testing.T, msg sdk.Msg, option ...Option) *codectypes.Any {
	t.Helper()

	// set options
	cfg := &Config{}
	for _, opt := range option {
		opt(cfg)
	}

	if cfg.AutomaticCommit {
		defer func() {
			_, err := app.Commit()
			require.NoError(t, err, "failed to commit")
			app.nextBlockCtx()
		}()
	}

	if cfg.AutomaticFinalizeBlock {
		height := app.LastBlockHeight() + 1
		_, err := app.FinalizeBlock(&cmtabcitypes.RequestFinalizeBlock{
			Height: height,
			DecidedLastCommit: cmtabcitypes.CommitInfo{
				Votes: []cmtabcitypes.VoteInfo{{}},
			},
		})
		require.NoError(t, err, "failed to finalize block")
	}

	app.Logger.Info("Running msg", "msg", msg.String())

	handler := app.MsgServiceRouter().Handler(msg)
	require.NotNil(t, handler, "handler not found for message %s", sdk.MsgTypeURL(msg))

	msgResult, err := handler(app.Ctx, msg)
	require.NoError(t, err, "failed to execute message %s", sdk.MsgTypeURL(msg))

	var response *codectypes.Any
	if len(msgResult.MsgResponses) > 0 {
		msgResponse := msgResult.MsgResponses[0]
		require.NotNil(t, msgResponse, "unexpected nil msg response %s in message result: %s", sdk.MsgTypeURL(msg), msgResult.String())
		response = msgResponse
	}

	return response
}

func (app *App) NextBlock(t *testing.T) {
	t.Helper()

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: app.Ctx.BlockHeight(),
		Time:   app.Ctx.BlockTime()})
	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	app.nextBlockCtx()
}

func (app *App) nextBlockCtx() {
	newBlockTime := app.Ctx.BlockTime().Add(time.Duration(1) * time.Second)

	header := app.Ctx.BlockHeader()
	header.Time = newBlockTime
	header.Height++

	newCtx := app.BaseApp.NewUncachedContext(false, header).
		WithHeaderInfo(coreheader.Info{
			Height: header.Height,
			Time:   header.Time,
		})

	app.Ctx = newCtx
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
