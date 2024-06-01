package integration

import (
	"fmt"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
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
	pooltypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	shared "github.com/pokt-network/poktroll/x/shared/module"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomics "github.com/pokt-network/poktroll/x/tokenomics/module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const appName = "integration-app"

func NewIntegrationApp(
	t *testing.T,
	sdkCtx sdk.Context,
	logger log.Logger,
	authority sdk.AccAddress,
	keys map[string]*storetypes.KVStoreKey,
	cdc codec.Codec,
	modules map[string]appmodule.AppModule,
	msgRouter *baseapp.MsgServiceRouter,
	grpcRouter *baseapp.GRPCQueryRouter,
) *App {
	t.Helper()

	db := dbm.NewMemDB()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	moduleManager := module.NewManagerFromMap(modules)
	basicModuleManager := module.NewBasicManagerFromManager(moduleManager, nil)
	basicModuleManager.RegisterInterfaces(interfaceRegistry)

	// configurator := module.NewConfigurator(cdc, msgRouter, grpcRouter)
	// moduleManager.RegisterServices(configurator)

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
	grpcRouter.SetInterfaceRegistry(interfaceRegistry)
	// bApp.S(grpcRouter)

	err := bApp.LoadLatestVersion()
	require.NoError(t, err, "failed to load latest version")

	_, err = bApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName})
	require.NoError(t, err, "failed to initialize chain")

	_, err = bApp.Commit()
	require.NoError(t, err, "failed to commit")

	cometHeader := cmtproto.Header{ChainID: appName}
	ctx := sdkCtx.WithBlockHeader(cometHeader).WithIsCheckTx(true)

	return &App{
		BaseApp:       bApp,
		Logger:        logger,
		Authority:     authority,
		Ctx:           ctx,
		Cdc:           cdc,
		ModuleManager: *moduleManager,
		QueryHelper:   baseapp.NewQueryServerTestHelper(ctx, interfaceRegistry),
	}
}

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

func NewCompleteIntegrationApp(t *testing.T) *App {
	t.Helper()

	// Register the codec for all the interfacesPrepare all the interfaces
	registry := codectypes.NewInterfaceRegistry()
	minttypes.RegisterInterfaces(registry)
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
		authtypes.StoreKey,
		minttypes.StoreKey,
		stakingtypes.StoreKey)

	// Prepare the context
	// logger := log.NewTestLogger(t)
	logger := log.NewNopLogger()
	cms := CreateMultiStore(storeKeys, logger)
	newCtx := sdk.NewContext(cms, cmtproto.Header{}, true, logger)

	// Get the authority address
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the account keeper
	addrCodec := addresscodec.NewBech32Codec(app.AccountAddressPrefix)
	macPerms := map[string][]string{
		pooltypes.ModuleName:           {},
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		banktypes.ModuleName:           {authtypes.Minter, authtypes.Burner},
		tokenomicstypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
		gatewaytypes.ModuleName:        {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		sessiontypes.ModuleName:        {authtypes.Minter, authtypes.Burner},
		apptypes.ModuleName:            {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		suppliertypes.ModuleName:       {authtypes.Minter, authtypes.Burner, authtypes.Staking},
	}

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

	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[stakingtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		authority.String(),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixValAddr),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)
	stakingModule := staking.NewAppModule(cdc, stakingKeeper, accountKeeper, bankKeeper, nil)

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

	proofKeeper := proofkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[prooftypes.StoreKey]),
		logger,
		authority.String(),
		nil, // sessionk
		applicationKeeper,
		accountKeeper,
		sharedKeeper,
	)
	proofModule := proof.NewAppModule(
		cdc,
		proofKeeper,
		accountKeeper,
	)

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

	msgRouter := baseapp.NewMsgServiceRouter()
	grpcRouter := baseapp.NewGRPCQueryRouter()

	modules := map[string]appmodule.AppModule{
		tokenomicstypes.ModuleName: tokenomicsModule,
		// banktypes.ModuleName:       bankModule,
		sharedtypes.ModuleName:  sharedModule,
		gatewaytypes.ModuleName: gatewayModule,
		// sessiontypes.ModuleName:    sessionModule,
		apptypes.ModuleName: applicationModule,
		// suppliertypes.ModuleName:   supplierModule,
		prooftypes.ModuleName: proofModule,
		authtypes.ModuleName:  authModule,
		// minttypes.ModuleName:    mintModule,
		stakingtypes.ModuleName: stakingModule,
	}

	integrationApp := NewIntegrationApp(
		t,
		newCtx,
		logger,
		authority,
		storeKeys,
		cdc,
		modules,
		msgRouter,
		grpcRouter,
	)

	authtypes.RegisterMsgServer(msgRouter, authkeeper.NewMsgServerImpl(accountKeeper))
	tokenomicstypes.RegisterMsgServer(msgRouter, tokenomicskeeper.NewMsgServerImpl(tokenomicsKeeper))

	// sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())

	return integrationApp
}

// RunMsg provides the ability to run a message and return the response.
// In order to run a message, the application must have a handler for it.
// These handlers are registered on the application message service router.
// The result of the message execution is returned as an Any type.
// That any type can be unmarshaled to the expected response type.
// If the message execution fails, an error is returned.
func (app *App) RunMsg(msg sdk.Msg, option ...Option) (*codectypes.Any, error) {
	// set options
	cfg := &Config{}
	for _, opt := range option {
		opt(cfg)
	}

	if cfg.AutomaticCommit {
		defer func() {
			_, err := app.Commit()
			if err != nil {
				panic(err)
			}
		}()
	}

	if cfg.AutomaticFinalizeBlock {
		height := app.LastBlockHeight() + 1
		if _, err := app.FinalizeBlock(&cmtabcitypes.RequestFinalizeBlock{
			Height: height,
			DecidedLastCommit: cmtabcitypes.CommitInfo{
				Votes: []cmtabcitypes.VoteInfo{{}},
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to run finalize block: %w", err)
		}
	}

	app.Logger.Info("Running msg", "msg", msg.String())

	handler := app.MsgServiceRouter().Handler(msg)
	if handler == nil {
		return nil, fmt.Errorf("handler is nil, can't route message %s: %+v", sdk.MsgTypeURL(msg), msg)
	}

	msgResult, err := handler(app.Ctx, msg)
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

// Context returns the application context. It can be unwrapped to a sdk.Context,
// with the sdk.UnwrapSDKContext function.
// func (app *App) Context() context.Context {
// 	return app.Ctx
// }

// QueryHelper returns the application query helper.
// It can be used when registering query services.
// func (app *App) QueryHelper() *baseapp.QueryServiceTestHelper {
// 	return app.queryHelper
// }

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
