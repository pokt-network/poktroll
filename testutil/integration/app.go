package integration

import (
	"testing"
	"time"

	"cosmossdk.io/core/appmodule"
	coreheader "cosmossdk.io/core/header"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
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
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	application "github.com/pokt-network/poktroll/x/application/module"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gateway "github.com/pokt-network/poktroll/x/gateway/module"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	proof "github.com/pokt-network/poktroll/x/proof/module"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	service "github.com/pokt-network/poktroll/x/service/module"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	session "github.com/pokt-network/poktroll/x/session/module"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	shared "github.com/pokt-network/poktroll/x/shared/module"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	supplier "github.com/pokt-network/poktroll/x/supplier/module"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomics "github.com/pokt-network/poktroll/x/tokenomics/module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const appName = "poktroll-integration-app"

// App is a test application that can be used to test the behaviour when none
// of the modules are mocked and their integration (cross module interaction)
// needs to be validated.
type App struct {
	*baseapp.BaseApp

	// Internal state of the App needed for properly configuring the blockchain.
	sdkCtx        *sdk.Context
	cdc           codec.Codec
	logger        log.Logger
	authority     sdk.AccAddress
	moduleManager module.Manager
	queryHelper   *baseapp.QueryServiceTestHelper
	keyRing       keyring.Keyring
	ringClient    crypto.RingClient

	// Some default helper fixtures for general testing.
	// They're publically exposed and should/could be improve and expand on
	// over time.
	DefaultService                   *sharedtypes.Service
	DefaultApplication               *apptypes.Application
	DefaultApplicationKeyringUid     string
	DefaultSupplier                  *sharedtypes.Supplier
	DefaultSupplierKeyringKeyringUid string
}

// NewIntegrationApp creates a new instance of the App with the provided details
// on how the modules should be configured.
func NewIntegrationApp(
	t *testing.T,
	sdkCtx sdk.Context,
	cdc codec.Codec,
	registry codectypes.InterfaceRegistry,
	logger log.Logger,
	authority sdk.AccAddress,
	modules map[string]appmodule.AppModule,
	keys map[string]*storetypes.KVStoreKey,
	msgRouter *baseapp.MsgServiceRouter,
	queryHelper *baseapp.QueryServiceTestHelper,
) *App {
	t.Helper()

	db := dbm.NewMemDB()

	moduleManager := module.NewManagerFromMap(modules)
	basicModuleManager := module.NewBasicManagerFromManager(moduleManager, nil)
	basicModuleManager.RegisterInterfaces(registry)

	// TODO_HACK(@Olshansk): I needed to set the height to 2 so downstream logic
	// works. I'm not 100% sure why, but believe it's a result of genesis and the
	// first block being special and iterated over during the setup process.
	cometHeader := cmtproto.Header{
		ChainID: appName,
		Height:  2,
	}
	sdkCtx = sdkCtx.
		WithBlockHeader(cometHeader).
		WithIsCheckTx(true).
		WithEventManager(cosmostypes.NewEventManager())

	// Add a block proposer address to the context
	valAddr, err := cosmostypes.ValAddressFromBech32(sample.ConsAddress())
	require.NoError(t, err)
	consensusAddr := cosmostypes.ConsAddress(valAddr)
	sdkCtx = sdkCtx.WithProposer(consensusAddr)

	// Create the base application
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
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

	msgRouter.SetInterfaceRegistry(registry)
	bApp.SetMsgServiceRouter(msgRouter)

	err = bApp.LoadLatestVersion()
	require.NoError(t, err, "failed to load latest version")

	_, err = bApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName})
	require.NoError(t, err, "failed to initialize chain")

	_, err = bApp.Commit()
	require.NoError(t, err, "failed to commit")

	return &App{
		BaseApp:       bApp,
		logger:        logger,
		authority:     authority,
		sdkCtx:        &sdkCtx,
		cdc:           cdc,
		moduleManager: *moduleManager,
		queryHelper:   queryHelper,
	}
}

// NewCompleteIntegrationApp creates a new instance of the App, abstracting out
// all of the internal details and complexities of the application setup.
// TODO_TECHDEBT: Not all of the modules are created here (e.g. minting module),
// so it is up to the developer to add / improve / update this function over time
// as the need arises.
func NewCompleteIntegrationApp(t *testing.T) *App {
	t.Helper()

	// Prepare & register the codec for all the interfaces
	registry := codectypes.NewInterfaceRegistry()
	tokenomicstypes.RegisterInterfaces(registry)
	sharedtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	gatewaytypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	sessiontypes.RegisterInterfaces(registry)
	apptypes.RegisterInterfaces(registry)
	suppliertypes.RegisterInterfaces(registry)
	prooftypes.RegisterInterfaces(registry)
	servicetypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	cosmostypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)

	// Prepare the codec
	cdc := codec.NewProtoCodec(registry)

	// Prepare all the store keys
	storeKeys := storetypes.NewKVStoreKeys(
		sharedtypes.StoreKey,
		tokenomicstypes.StoreKey,
		banktypes.StoreKey,
		gatewaytypes.StoreKey,
		sessiontypes.StoreKey,
		apptypes.StoreKey,
		suppliertypes.StoreKey,
		prooftypes.StoreKey,
		servicetypes.StoreKey,
		authtypes.StoreKey,
	)

	// Prepare the context
	logger := log.NewNopLogger() // Use this if you need more output: log.NewTestLogger(t)
	cms := CreateMultiStore(storeKeys, logger)
	sdkCtx := sdk.NewContext(cms, cmtproto.Header{
		ChainID: appName,
		Height:  1,
	}, true, logger)

	// Get the authority address
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the account keeper dependencies
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

	// Prepare the bank keeper
	blockedAddresses := map[string]bool{
		accountKeeper.GetAuthority(): false,
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[banktypes.StoreKey]),
		accountKeeper,
		blockedAddresses,
		authority.String(),
		logger,
	)

	// Prepare the shared keeper and module
	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[sharedtypes.StoreKey]),
		logger,
		authority.String(),
	)
	sharedModule := shared.NewAppModule(
		cdc,
		sharedKeeper,
		accountKeeper,

		bankKeeper,
	)

	// Prepare the service keeper and module
	serviceKeeper := servicekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[servicetypes.StoreKey]),
		logger,
		authority.String(),

		bankKeeper,
	)
	serviceModule := service.NewAppModule(
		cdc,
		serviceKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the gateway keeper and module
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[gatewaytypes.StoreKey]),
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

	// Prepare the supplier keeper and module
	supplierKeeper := supplierkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[suppliertypes.StoreKey]),
		logger,
		authority.String(),

		bankKeeper,
		sharedKeeper,
		serviceKeeper,
	)
	supplierModule := supplier.NewAppModule(
		cdc,
		supplierKeeper,
		accountKeeper,
		bankKeeper,
		serviceKeeper,
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
		supplierKeeper,
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
		supplierKeeper,
		proofKeeper,
		sharedKeeper,
		sessionKeeper,
		serviceKeeper,
	)
	tokenomicsModule := tokenomics.NewAppModule(
		cdc,
		tokenomicsKeeper,
		accountKeeper,
		bankKeeper,
	)

	// Prepare the message & query routers
	msgRouter := baseapp.NewMsgServiceRouter()
	queryHelper := baseapp.NewQueryServerTestHelper(sdkCtx, registry)

	// Prepare the list of modules
	modules := map[string]appmodule.AppModule{
		tokenomicstypes.ModuleName: tokenomicsModule,
		servicetypes.ModuleName:    serviceModule,
		sharedtypes.ModuleName:     sharedModule,
		gatewaytypes.ModuleName:    gatewayModule,
		apptypes.ModuleName:        applicationModule,
		suppliertypes.ModuleName:   supplierModule,
		prooftypes.ModuleName:      proofModule,
		authtypes.ModuleName:       authModule,
		sessiontypes.ModuleName:    sessionModule,
	}

	// Initialize the integration integrationApp
	integrationApp := NewIntegrationApp(
		t,
		sdkCtx,
		cdc,
		registry,
		logger,
		authority,
		modules,
		storeKeys,
		msgRouter,
		queryHelper,
	)

	// Register the message servers
	tokenomicstypes.RegisterMsgServer(msgRouter, tokenomicskeeper.NewMsgServerImpl(tokenomicsKeeper))
	servicetypes.RegisterMsgServer(msgRouter, servicekeeper.NewMsgServerImpl(serviceKeeper))
	sharedtypes.RegisterMsgServer(msgRouter, sharedkeeper.NewMsgServerImpl(sharedKeeper))
	gatewaytypes.RegisterMsgServer(msgRouter, gatewaykeeper.NewMsgServerImpl(gatewayKeeper))
	apptypes.RegisterMsgServer(msgRouter, appkeeper.NewMsgServerImpl(applicationKeeper))
	suppliertypes.RegisterMsgServer(msgRouter, supplierkeeper.NewMsgServerImpl(supplierKeeper))
	prooftypes.RegisterMsgServer(msgRouter, proofkeeper.NewMsgServerImpl(proofKeeper))
	authtypes.RegisterMsgServer(msgRouter, authkeeper.NewMsgServerImpl(accountKeeper))
	sessiontypes.RegisterMsgServer(msgRouter, sessionkeeper.NewMsgServerImpl(sessionKeeper))

	// Register query servers
	tokenomicstypes.RegisterQueryServer(queryHelper, tokenomicsKeeper)
	servicetypes.RegisterQueryServer(queryHelper, serviceKeeper)
	sharedtypes.RegisterQueryServer(queryHelper, sharedKeeper)
	gatewaytypes.RegisterQueryServer(queryHelper, gatewayKeeper)
	apptypes.RegisterQueryServer(queryHelper, applicationKeeper)
	suppliertypes.RegisterQueryServer(queryHelper, supplierKeeper)
	prooftypes.RegisterQueryServer(queryHelper, proofKeeper)
	// TODO_TECHDEBT: What is the query server for authtypes?
	// authtypes.RegisterQueryServer(queryHelper, accountKeeper)
	sessiontypes.RegisterQueryServer(queryHelper, sessionKeeper)

	// Set the default params for all the modules
	err := sharedKeeper.SetParams(integrationApp.GetSdkCtx(), sharedtypes.DefaultParams())
	require.NoError(t, err)
	err = tokenomicsKeeper.SetParams(integrationApp.GetSdkCtx(), tokenomicstypes.DefaultParams())
	require.NoError(t, err)
	err = proofKeeper.SetParams(integrationApp.GetSdkCtx(), prooftypes.DefaultParams())
	require.NoError(t, err)
	err = sessionKeeper.SetParams(integrationApp.GetSdkCtx(), sessiontypes.DefaultParams())
	require.NoError(t, err)
	err = gatewayKeeper.SetParams(integrationApp.GetSdkCtx(), gatewaytypes.DefaultParams())
	require.NoError(t, err)
	err = applicationKeeper.SetParams(integrationApp.GetSdkCtx(), apptypes.DefaultParams())
	require.NoError(t, err)

	// Need to go to the next block to finalize the genesis and setup.
	// This has to be after the params are set, as the params are stored in the
	// store and need to be committed.
	integrationApp.NextBlock(t)

	// Prepare default testing fixtures //

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(integrationApp.cdc)
	integrationApp.keyRing = keyRing

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Prepare a new default service
	defaultService := sharedtypes.Service{
		Id:           "svc1",
		Name:         "svcName1",
		OwnerAddress: sample.AccAddress(),
	}
	serviceKeeper.SetService(integrationApp.sdkCtx, defaultService)
	integrationApp.DefaultService = &defaultService

	// Create a supplier account with the corresponding keys in the keyring for the supplier.
	integrationApp.DefaultSupplierKeyringKeyringUid = "supplier"
	supplierAddr := testkeyring.CreateOnChainAccount(
		integrationApp.sdkCtx, t,
		integrationApp.DefaultSupplierKeyringKeyringUid,
		keyRing,
		accountKeeper,
		preGeneratedAccts,
	)

	// Prepare the on-chain supplier
	supplierStake := types.NewCoin("upokt", math.NewInt(1000000))
	defaultSupplier := sharedtypes.Supplier{
		Address: supplierAddr.String(),
		Stake:   &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				RevShare: []*sharedtypes.ServiceRevShare{
					{
						Address:            sample.AccAddress(),
						RevSharePercentage: 100,
					},
				},
				Service: &defaultService,
			},
		},
	}
	supplierKeeper.SetSupplier(integrationApp.sdkCtx, defaultSupplier)
	integrationApp.DefaultSupplier = &defaultSupplier

	// Create an application account with the corresponding keys in the keyring for the application.
	integrationApp.DefaultApplicationKeyringUid = "application"
	applicationAddr := testkeyring.CreateOnChainAccount(
		integrationApp.sdkCtx, t,
		integrationApp.DefaultApplicationKeyringUid,
		keyRing,
		accountKeeper,
		preGeneratedAccts,
	)

	// Prepare the on-chain supplier
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	defaultApplication := apptypes.Application{
		Address: applicationAddr.String(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &defaultService,
			},
		},
	}
	applicationKeeper.SetApplication(integrationApp.sdkCtx, defaultApplication)
	integrationApp.DefaultApplication = &defaultApplication

	// Construct a ringClient to get the application's ring & verify the relay
	// request signature.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		prooftypes.NewAppKeeperQueryClient(applicationKeeper),
		prooftypes.NewAccountKeeperQueryClient(accountKeeper),
		prooftypes.NewSharedKeeperQueryClient(sharedKeeper, sessionKeeper),
	))
	require.NoError(t, err)
	integrationApp.ringClient = ringClient

	// TODO_IMPROVE: The setup above does not to proper "staking" of the suppliers and applications.
	// This can result in the module accounts balance going negative. Giving them a baseline balance
	// to start with to avoid this issue. There is opportunity to improve this in the future.
	moduleBaseMint := types.NewCoins(sdk.NewCoin("upokt", math.NewInt(690000000000000042)))
	err = bankKeeper.MintCoins(integrationApp.sdkCtx, suppliertypes.ModuleName, moduleBaseMint)
	require.NoError(t, err)
	err = bankKeeper.MintCoins(integrationApp.sdkCtx, apptypes.ModuleName, moduleBaseMint)
	require.NoError(t, err)

	// Commit all the changes above by committing, finalizing and moving
	// to the next block.
	integrationApp.NextBlock(t)

	return integrationApp
}

// GetRingClient returns the ring client used by the application.
func (app *App) GetRingClient() crypto.RingClient {
	return app.ringClient
}

// GetKeyRing returns the keyring used by the application.
func (app *App) GetKeyRing() keyring.Keyring {
	return app.keyRing
}

// GetCodec returns the codec used by the application.
func (app *App) GetCodec() codec.Codec {
	return app.cdc
}

// GetSdkCtx returns the context used by the application.
func (app *App) GetSdkCtx() *sdk.Context {
	return app.sdkCtx
}

// GetAuthority returns the authority address used by the application.
func (app *App) GetAuthority() string {
	return app.authority.String()
}

// QueryHelper returns the query helper used by the application that can be
// used to submit queries to the application.
func (app *App) QueryHelper() *baseapp.QueryServiceTestHelper {
	app.queryHelper.Ctx = *app.sdkCtx
	return app.queryHelper
}

// RunMsg provides the ability to run a message and return the response.
// In order to run a message, the application must have a handler for it.
// These handlers are registered on the application message service router.
// The result of the message execution is returned as an Any type.
// That any type can be unmarshaled to the expected response type.
// If the message execution fails, an error is returned.
func (app *App) RunMsg(t *testing.T, msg sdk.Msg, option ...RunOption) *codectypes.Any {
	t.Helper()

	// set options
	cfg := &RunConfig{}
	for _, opt := range option {
		opt(cfg)
	}

	// If configured, commit after the message is executed.
	if cfg.AutomaticCommit {
		defer func() {
			_, err := app.Commit()
			require.NoError(t, err, "failed to commit")
			app.nextBlockUpdateCtx()
		}()
	}

	// If configured, finalize the block after the message is executed.
	if cfg.AutomaticFinalizeBlock {
		finalizedBlockResponse, err := app.FinalizeBlock(&cmtabcitypes.RequestFinalizeBlock{
			Height: app.LastBlockHeight() + 1,
			DecidedLastCommit: cmtabcitypes.CommitInfo{
				Votes: []cmtabcitypes.VoteInfo{{}},
			},
		})
		require.NoError(t, err, "failed to finalize block")
		app.emitEvents(t, finalizedBlockResponse)
	}

	app.logger.Info("Running msg", "msg", msg.String())

	handler := app.MsgServiceRouter().Handler(msg)
	require.NotNil(t, handler, "handler not found for message %s", sdk.MsgTypeURL(msg))

	msgResult, err := handler(*app.sdkCtx, msg)
	require.NoError(t, err, "failed to execute message %s", sdk.MsgTypeURL(msg))

	var response *codectypes.Any
	if len(msgResult.MsgResponses) > 0 {
		msgResponse := msgResult.MsgResponses[0]
		require.NotNil(t, msgResponse, "unexpected nil msg response %s in message result: %s", sdk.MsgTypeURL(msg), msgResult.String())
		response = msgResponse
	}

	return response
}

// NextBlocks calls NextBlock numBlocks times
func (app *App) NextBlocks(t *testing.T, numBlocks int) {
	t.Helper()

	for i := 0; i < numBlocks; i++ {
		app.NextBlock(t)
	}
}

// emitEvents emits the events from the finalized block to the event manager
// of the context in the active app.
func (app *App) emitEvents(t *testing.T, res *abci.ResponseFinalizeBlock) {
	t.Helper()
	for _, event := range res.Events {
		testutilevents.QuoteEventMode(&event)
		abciEvent := cosmostypes.Event(event)
		app.sdkCtx.EventManager().EmitEvent(abciEvent)
	}
}

// NextBlock commits and finalizes all existing transactions. It then updates
// and advances the context of the App.
func (app *App) NextBlock(t *testing.T) {
	t.Helper()

	finalizedBlockResponse, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: app.sdkCtx.BlockHeight(),
		Time:   app.sdkCtx.BlockTime()})
	require.NoError(t, err)
	app.emitEvents(t, finalizedBlockResponse)

	_, err = app.Commit()
	require.NoError(t, err)

	app.nextBlockUpdateCtx()
}

// nextBlockUpdateCtx is responsible for updating the app's (receiver) context
// to the next block. It does not trigger ABCI specific business logic but manages
// app.sdkCtx related metadata so downstream queries and transactions are executed
// in the correct context.
func (app *App) nextBlockUpdateCtx() {
	prevCtx := app.sdkCtx

	header := prevCtx.BlockHeader()
	header.Time = prevCtx.BlockTime().Add(time.Duration(1) * time.Second)
	header.Height++

	headerInfo := coreheader.Info{
		ChainID: appName,
		Height:  header.Height,
		Time:    header.Time,
	}

	newContext := app.BaseApp.NewUncachedContext(true, header).
		WithBlockHeader(header).
		WithHeaderInfo(headerInfo).
		WithEventManager(prevCtx.EventManager()).
		// Pass the multi-store to the new context, otherwise the new context will
		// create a new multi-store.
		WithMultiStore(prevCtx.MultiStore())
	*app.sdkCtx = newContext
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
