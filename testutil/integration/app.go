package integration

import (
	"context"
	"errors"
	"fmt"
	math2 "math"
	"testing"
	"time"

	"cosmossdk.io/core/appmodule"
	coreheader "cosmossdk.io/core/header"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtabcitypes "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
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
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/app/volatile"
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

var (
	// faucetAmountUpokt is the number of upokt coins that the faucet account
	// is funded with.
	faucetAmountUpokt = int64(math2.MaxInt64)
)

// App is a test application that can be used to test the behaviour when none
// of the modules are mocked and their integration (cross module interaction)
// needs to be validated.
type App struct {
	*baseapp.BaseApp

	// Internal state of the App needed for properly configuring the blockchain.
	sdkCtx            *sdk.Context
	cdc               codec.Codec
	logger            log.Logger
	txCfg             client.TxConfig
	authority         sdk.AccAddress
	moduleManager     module.Manager
	queryHelper       *baseapp.QueryServiceTestHelper
	keyRing           keyring.Keyring
	ringClient        crypto.RingClient
	preGeneratedAccts *testkeyring.PreGeneratedAccountIterator

	// faucetBech32 is a random address which is selected as the primary faucet
	// to fund other accounts. It is funded with faucetAmountUpokt coins such that
	// it can be used as a faucet for integration tests.
	faucetBech32 string

	// Some default helper fixtures for general testing.
	// They're publicly exposed and should/could be improved and expand on
	// over time.
	//
	// TODO_IMPROVE: Refactor into a DefaultActorsIntegrationSuite test suite.
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
	txCfg client.TxConfig,
	registry codectypes.InterfaceRegistry,
	bApp *baseapp.BaseApp,
	logger log.Logger,
	authority sdk.AccAddress,
	modules map[string]appmodule.AppModule,
	keys map[string]*storetypes.KVStoreKey,
	msgRouter *baseapp.MsgServiceRouter,
	queryHelper *baseapp.QueryServiceTestHelper,
	opts ...IntegrationAppOptionFn,
) *App {
	t.Helper()

	// Prepare the faucet init-chainer module option function. It ensures that the
	// bank module genesis state includes the faucet account with a large balance.
	faucetBech32 := sample.AccAddress()
	faucetInitChainerFn := newFaucetInitChainerFn(faucetBech32, faucetAmountUpokt)
	initChainerModuleOptFn := WithInitChainerModuleFn(faucetInitChainerFn)

	cfg := &IntegrationAppConfig{}
	opts = append(opts, initChainerModuleOptFn)
	for _, opt := range opts {
		opt(cfg)
	}

	bApp.SetInterfaceRegistry(registry)

	moduleManager := module.NewManagerFromMap(modules)
	basicModuleManager := module.NewBasicManagerFromManager(moduleManager, nil)
	basicModuleManager.RegisterInterfaces(registry)

	cometHeader := cmtproto.Header{
		ChainID: appName,
		Height:  1,
	}
	sdkCtx = sdkCtx.
		WithBlockHeader(cometHeader).
		WithIsCheckTx(true).
		WithEventManager(cosmostypes.NewEventManager())

	// Add a block proposer address to the context
	valAddr, err := cosmostypes.ValAddressFromBech32(sample.ValAddress())
	require.NoError(t, err)
	consensusAddr := cosmostypes.ConsAddress(valAddr)
	sdkCtx = sdkCtx.WithProposer(consensusAddr)

	// Create the base application
	bApp.MountKVStores(keys)

	bApp.SetInitChainer(
		func(ctx sdk.Context, _ *cmtabcitypes.RequestInitChain) (*cmtabcitypes.ResponseInitChain, error) {
			for _, mod := range modules {
				// Set each module's genesis state to the default. This MAY be
				// overridden via the InitChainerModuleFns option.
				if m, ok := mod.(module.HasGenesis); ok {
					m.InitGenesis(ctx, cdc, m.DefaultGenesis(cdc))
				}

				// Call each of the InitChainerModuleFns for each module.
				for _, fn := range cfg.InitChainerModuleFns {
					fn(ctx, cdc, mod)
				}
			}

			return &cmtabcitypes.ResponseInitChain{}, nil
		})

	bApp.SetBeginBlocker(func(ctx sdk.Context) (sdk.BeginBlock, error) {
		return moduleManager.BeginBlock(ctx)
	})
	bApp.SetEndBlocker(func(ctx sdk.Context) (sdk.EndBlock, error) {
		return moduleManager.EndBlock(ctx)
	})

	msgRouter.SetInterfaceRegistry(registry)
	bApp.SetMsgServiceRouter(msgRouter)

	err = bApp.LoadLatestVersion()
	require.NoError(t, err, "failed to load latest version")

	_, err = bApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName})
	require.NoError(t, err, "failed to initialize chain")

	bApp.SetTxEncoder(txCfg.TxEncoder())

	return &App{
		BaseApp:       bApp,
		logger:        logger,
		authority:     authority,
		sdkCtx:        &sdkCtx,
		cdc:           cdc,
		txCfg:         txCfg,
		moduleManager: *moduleManager,
		queryHelper:   queryHelper,
		faucetBech32:  faucetBech32,
	}
}

// NewCompleteIntegrationApp creates a new instance of the App, abstracting out
// all the internal details and complexities of the application setup.
//
// TODO_TECHDEBT: Not all of the modules are created here (e.g. minting module),
// so it is up to the developer to add / improve / update this function over time
// as the need arises.
func NewCompleteIntegrationApp(t *testing.T, opts ...IntegrationAppOptionFn) *App {
	t.Helper()

	// Prepare & register the codec for all the interfaces
	sdkCfg := cosmostypes.GetConfig()
	addrCodec := addresscodec.NewBech32Codec(sdkCfg.GetBech32AccountAddrPrefix())
	valCodec := addresscodec.NewBech32Codec(sdkCfg.GetBech32ValidatorAddrPrefix())
	signingOpts := signing.Options{
		AddressCodec:          addrCodec,
		ValidatorAddressCodec: valCodec,
	}
	registryOpts := codectypes.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOpts,
	}
	registry, err := codectypes.NewInterfaceRegistryWithOptions(registryOpts)
	require.NoError(t, err)

	banktypes.RegisterInterfaces(registry)
	authz.RegisterInterfaces(registry)
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

	// Prepare all the store keys
	storeKeys := storetypes.NewKVStoreKeys(
		authzkeeper.StoreKey,
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

	// Prepare the codec
	cdc := codec.NewProtoCodec(registry)

	// Prepare the TxConfig
	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	// Construct a no-op logger.
	logger := log.NewNopLogger() // Use this if you need more output: log.NewTestLogger(t)

	// Prepare the database and multi-store.
	db := dbm.NewMemDB()

	// Prepare the base application.
	bApp := baseapp.NewBaseApp(appName, logger, db, txCfg.TxDecoder(), baseapp.SetChainID(appName))

	// Prepare the context
	sdkCtx := bApp.NewUncachedContext(false, cmtproto.Header{
		ChainID: appName,
		Height:  1,
	})

	// Get the authority address
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the account keeper dependencies
	macPerms := map[string][]string{
		banktypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		tokenomicstypes.ModuleName: {authtypes.Minter, authtypes.Burner},
		gatewaytypes.ModuleName:    {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		sessiontypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		apptypes.ModuleName:        {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		suppliertypes.ModuleName:   {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		prooftypes.ModuleName:      {authtypes.Minter, authtypes.Burner},
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

	// Prepare the bank keeper and module
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
	bankModule := bank.NewAppModule(
		cdc,
		bankKeeper,
		accountKeeper,
		nil,
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

		bankKeeper,
		sessionKeeper,
		applicationKeeper,
		accountKeeper,
		sharedKeeper,
		serviceKeeper,
	)
	proofModule := proof.NewAppModule(
		cdc,
		proofKeeper,
		accountKeeper,
		bankKeeper,
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
		supplierKeeper,
	)

	// Prepare the message & query routers
	msgRouter := baseapp.NewMsgServiceRouter()
	queryHelper := baseapp.NewQueryServerTestHelper(sdkCtx, registry)

	// Prepare the authz keeper and module
	authzKeeper := authzkeeper.NewKeeper(
		runtime.NewKVStoreService(storeKeys[authzkeeper.StoreKey]),
		cdc,
		msgRouter,
		accountKeeper,
	).SetBankKeeper(bankKeeper)
	authzModule := authzmodule.NewAppModule(
		cdc,
		authzKeeper,
		accountKeeper,
		bankKeeper,
		registry,
	)

	// Prepare the list of modules
	modules := map[string]appmodule.AppModule{
		banktypes.ModuleName:       bankModule,
		authz.ModuleName:           authzModule,
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
		txCfg,
		registry,
		bApp,
		logger,
		authority,
		modules,
		storeKeys,
		msgRouter,
		queryHelper,
		opts...,
	)

	// Register the message servers
	banktypes.RegisterMsgServer(msgRouter, bankkeeper.NewMsgServerImpl(bankKeeper))
	tokenomicstypes.RegisterMsgServer(msgRouter, tokenomicskeeper.NewMsgServerImpl(tokenomicsKeeper))
	servicetypes.RegisterMsgServer(msgRouter, servicekeeper.NewMsgServerImpl(serviceKeeper))
	sharedtypes.RegisterMsgServer(msgRouter, sharedkeeper.NewMsgServerImpl(sharedKeeper))
	gatewaytypes.RegisterMsgServer(msgRouter, gatewaykeeper.NewMsgServerImpl(gatewayKeeper))
	apptypes.RegisterMsgServer(msgRouter, appkeeper.NewMsgServerImpl(applicationKeeper))
	suppliertypes.RegisterMsgServer(msgRouter, supplierkeeper.NewMsgServerImpl(supplierKeeper))
	prooftypes.RegisterMsgServer(msgRouter, proofkeeper.NewMsgServerImpl(proofKeeper))
	authtypes.RegisterMsgServer(msgRouter, authkeeper.NewMsgServerImpl(accountKeeper))
	sessiontypes.RegisterMsgServer(msgRouter, sessionkeeper.NewMsgServerImpl(sessionKeeper))
	authz.RegisterMsgServer(msgRouter, authzKeeper)

	// Register query servers
	banktypes.RegisterQueryServer(queryHelper, bankKeeper)
	authz.RegisterQueryServer(queryHelper, authzKeeper)
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
	integrationApp.preGeneratedAccts = preGeneratedAccts

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

	// TODO_IMPROVE: Eliminate usage of and remove this function in favor of
	// integration.NewInitChainerModuleGenesisStateOptionFn.
	integrationApp.setupDefaultActorsState(t,
		accountKeeper,
		bankKeeper,
		serviceKeeper,
		supplierKeeper,
		applicationKeeper,
	)

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

// GetPreGeneratedAccounts returns the pre-generated accounts iterater used by the application.
func (app *App) GetPreGeneratedAccounts() *testkeyring.PreGeneratedAccountIterator {
	return app.preGeneratedAccts
}

// QueryHelper returns the query helper used by the application that can be
// used to submit queries to the application.
func (app *App) QueryHelper() *baseapp.QueryServiceTestHelper {
	app.queryHelper.Ctx = *app.sdkCtx
	return app.queryHelper
}

// RunMsg provides the ability to process a message by packing it into a tx and
// driving the ABCI through block finalization. It returns a tx.MsgResponse (any)
// which corresponds to the request message. It is a convenience method which wraps
// RunMsgs.
func (app *App) RunMsg(t *testing.T, msg sdk.Msg) (tx.MsgResponse, error) {
	t.Helper()

	txMsgRes, err := app.RunMsgs(t, msg)
	if err != nil {
		return nil, err
	}

	require.Equal(t, 1, len(txMsgRes), "expected exactly 1 tx msg response")
	return txMsgRes[0], err
}

// GetFaucetBech32 returns the faucet address used by the application.
func (app *App) GetFaucetBech32() string {
	return app.faucetBech32
}

// RunMsgs provides the ability to process messages by packing them into a tx and
// driving the ABCI through block finalization. It returns a slice of tx.MsgResponse
// (any) whose elements correspond to the request message of the same index.
// These responses can be type asserted to the expected response type.
// If execution for ANY message fails, ALL failing messages' errors are joined and
// returned. In order to run a message, the application must have a handler for it.
// These handlers are registered on the application message service router.
func (app *App) RunMsgs(t *testing.T, msgs ...sdk.Msg) (txMsgResps []tx.MsgResponse, err error) {
	t.Helper()

	// Commit the updated state after the message has been handled.
	var finalizeBlockRes *abci.ResponseFinalizeBlock
	defer func() {
		if _, commitErr := app.Commit(); commitErr != nil {
			err = fmt.Errorf("committing state: %w", commitErr)
			return
		}

		app.nextBlockUpdateCtx()

		// Emit events MUST happen AFTER the context has been updated so that
		// events are available on the context for the block after their actions
		// were committed (e.g. msgs, begin/end block trigger).
		app.emitEvents(t, finalizeBlockRes)
	}()

	// Package the message into a transaction.
	txBuilder := app.txCfg.NewTxBuilder()
	if err = txBuilder.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("setting tx messages: %w", err)
	}

	txBz, txErr := app.TxEncode(txBuilder.GetTx())
	if txErr != nil {
		return nil, fmt.Errorf("encoding tx: %w", err)
	}

	for _, msg := range msgs {
		app.logger.Info("Running msg", "msg", msg.String())
	}

	// Finalize the block with the transaction.
	finalizeBlockReq := &cmtabcitypes.RequestFinalizeBlock{
		Height: app.LastBlockHeight() + 1,
		// Randomize the proposer address for each block.
		ProposerAddress: newProposerAddrBz(t),
		DecidedLastCommit: cmtabcitypes.CommitInfo{
			Votes: []cmtabcitypes.VoteInfo{{}},
		},
		Txs: [][]byte{txBz},
	}

	finalizeBlockRes, err = app.FinalizeBlock(finalizeBlockReq)
	if err != nil {
		return nil, fmt.Errorf("finalizing block: %w", err)
	}

	// NB: We're batching the messages in a single transaction, so we expect
	// a single transaction result.
	require.Equal(t, 1, len(finalizeBlockRes.TxResults))

	// Collect the message responses. Accumulate errors related to message handling
	// failure. If any message fails, an error will be returned.
	var txResultErrs error
	for _, txResult := range finalizeBlockRes.TxResults {
		if !txResult.IsOK() {
			err = fmt.Errorf("tx failed with log: %q", txResult.GetLog())
			txResultErrs = errors.Join(txResultErrs, err)
			continue
		}

		txMsgDataBz := txResult.GetData()
		require.NotNil(t, txMsgDataBz)

		txMsgData := new(cosmostypes.TxMsgData)
		err = app.GetCodec().Unmarshal(txMsgDataBz, txMsgData)
		require.NoError(t, err)

		var txMsgRes tx.MsgResponse
		err = app.GetCodec().UnpackAny(txMsgData.MsgResponses[0], &txMsgRes)
		require.NoError(t, err)
		require.NotNil(t, txMsgRes)

		txMsgResps = append(txMsgResps, txMsgRes)
	}
	if txResultErrs != nil {
		return nil, err
	}

	return txMsgResps, nil
}

// NextBlocks calls NextBlock numBlocks times
func (app *App) NextBlocks(t *testing.T, numBlocks int) {
	t.Helper()

	for i := 0; i < numBlocks; i++ {
		app.NextBlock(t)
	}
}

// emitEvents emits the events from the finalized block such that they are available
// via the current context's event manager (i.e. app.GetSdkCtx().EventManager.Events()).
func (app *App) emitEvents(t *testing.T, res *abci.ResponseFinalizeBlock) {
	t.Helper()

	// Emit begin/end blocker events.
	for _, event := range res.Events {
		testutilevents.QuoteEventMode(&event)
		abciEvent := cosmostypes.Event(event)
		app.sdkCtx.EventManager().EmitEvent(abciEvent)
	}

	// Emit txResult events.
	for _, txResult := range res.TxResults {
		for _, event := range txResult.Events {
			abciEvent := cosmostypes.Event(event)
			app.sdkCtx.EventManager().EmitEvent(abciEvent)
		}
	}
}

// NextBlock commits and finalizes all existing transactions. It then updates
// and advances the context of the App.
func (app *App) NextBlock(t *testing.T) {
	t.Helper()

	finalizedBlockResponse, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: app.sdkCtx.BlockHeight(),
		Time:   app.sdkCtx.BlockTime(),
		// Randomize the proposer address for each block.
		ProposerAddress: newProposerAddrBz(t),
	})
	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	app.nextBlockUpdateCtx()

	// Emit events MUST happen AFTER the context has been updated so that
	// events are available on the context for the block after their actions
	// were committed (e.g. msgs, begin/end block trigger).
	app.emitEvents(t, finalizedBlockResponse)
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

	// NB: Intentionally omitting the previous contexts EventManager.
	// This ensures that each event is only observed for 1 block.
	newContext := app.NewUncachedContext(true, header).
		WithBlockHeader(header).
		WithHeaderInfo(headerInfo).
		// Pass the multi-store to the new context, otherwise the new context will
		// create a new multi-store.
		WithMultiStore(prevCtx.MultiStore())
	*app.sdkCtx = newContext
}

// setupDefaultActorsState uses the integration app keepers to stake "default"
// on-chain actors for use in tests. In creates a service, and stakes a supplier
// and application as well as funding the bank balance of the default supplier.
//
// TODO_TECHDEBT(@bryanchriswhite): Eliminate usage of and remove this function in favor of
// integration.NewInitChainerModuleGenesisStateOptionFn.
func (app *App) setupDefaultActorsState(
	t *testing.T,
	accountKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	serviceKeeper servicekeeper.Keeper,
	supplierKeeper supplierkeeper.Keeper,
	applicationKeeper appkeeper.Keeper,
) {
	t.Helper()

	// Prepare a new default service
	defaultService := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}
	serviceKeeper.SetService(app.sdkCtx, defaultService)
	app.DefaultService = &defaultService

	// Create a supplier account with the corresponding keys in the keyring for the supplier.
	app.DefaultSupplierKeyringKeyringUid = "supplier"
	supplierOperatorAddr := testkeyring.CreateOnChainAccount(
		app.sdkCtx, t,
		app.DefaultSupplierKeyringKeyringUid,
		app.keyRing,
		accountKeeper,
		app.preGeneratedAccts,
	)

	// Prepare the on-chain supplier
	supplierStake := types.NewCoin("upokt", math.NewInt(1000000))
	defaultSupplier := sharedtypes.Supplier{
		OwnerAddress:    supplierOperatorAddr.String(),
		OperatorAddress: supplierOperatorAddr.String(),
		Stake:           &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            sample.AccAddress(),
						RevSharePercentage: 100,
					},
				},
				ServiceId: defaultService.Id,
			},
		},
	}
	supplierKeeper.SetSupplier(app.sdkCtx, defaultSupplier)
	app.DefaultSupplier = &defaultSupplier

	// Create an application account with the corresponding keys in the keyring for the application.
	app.DefaultApplicationKeyringUid = "application"
	applicationAddr := testkeyring.CreateOnChainAccount(
		app.sdkCtx, t,
		app.DefaultApplicationKeyringUid,
		app.keyRing,
		accountKeeper,
		app.preGeneratedAccts,
	)

	// Prepare the on-chain application
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	defaultApplication := apptypes.Application{
		Address: applicationAddr.String(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: defaultService.Id,
			},
		},
	}
	applicationKeeper.SetApplication(app.sdkCtx, defaultApplication)
	app.DefaultApplication = &defaultApplication

	// TODO_IMPROVE: The setup above does not to proper "staking" of the suppliers and applications.
	// This can result in the module accounts balance going negative. Giving them a baseline balance
	// to start with to avoid this issue. There is opportunity to improve this in the future.
	moduleBaseMint := types.NewCoins(sdk.NewCoin("upokt", math.NewInt(690000000000000042)))
	err := bankKeeper.MintCoins(app.sdkCtx, suppliertypes.ModuleName, moduleBaseMint)
	require.NoError(t, err)
	err = bankKeeper.MintCoins(app.sdkCtx, apptypes.ModuleName, moduleBaseMint)
	require.NoError(t, err)

	// TODO_IMPROVE: Refactor the relay_mining_difficulty_test.go to use the
	// BaseIntegrationTestSuite (or a DefaultActorIntegrationSuite) and its
	// #FundAddress() method and remove the need for this.
	//
	// Fund the supplier operator account to be able to submit proofs
	fundAccount(t, app.sdkCtx, bankKeeper, supplierOperatorAddr, 100000000)

	// Commit all the changes above by finalizing, committing, and moving
	// to the next block.
	app.NextBlock(t)
}

// fundAccount mints and sends amountUpokt tokens to the given recipientAddr.
//
// TODO_IMPROVE: Eliminate usage of and remove this function in favor of
// integration.NewInitChainerModuleGenesisStateOptionFn.
func fundAccount(
	t *testing.T,
	ctx context.Context,
	bankKeeper bankkeeper.Keeper,
	recipientAddr sdk.AccAddress,
	amountUpokt int64,
) {

	fundingCoins := types.NewCoins(types.NewCoin(volatile.DenomuPOKT, math.NewInt(amountUpokt)))

	err := bankKeeper.MintCoins(ctx, banktypes.ModuleName, fundingCoins)
	require.NoError(t, err)

	err = bankKeeper.SendCoinsFromModuleToAccount(ctx, banktypes.ModuleName, recipientAddr, fundingCoins)
	require.NoError(t, err)

	coin := bankKeeper.SpendableCoin(ctx, recipientAddr, volatile.DenomuPOKT)
	require.Equal(t, coin.Amount, math.NewInt(amountUpokt))
}

// newFaucetInitChainerFn returns an InitChainerModuleFn that initializes the bank module
// with a genesis state which contains a faucet account that has faucetAmtUpokt coins.
func newFaucetInitChainerFn(faucetBech32 string, faucetAmtUpokt int64) InitChainerModuleFn {
	return NewInitChainerModuleGenesisStateOptionFn[bank.AppModule](&banktypes.GenesisState{
		Params: banktypes.DefaultParams(),
		Balances: []banktypes.Balance{
			{
				Address: faucetBech32,
				Coins: sdk.NewCoins(
					sdk.NewInt64Coin(
						volatile.DenomuPOKT,
						faucetAmtUpokt,
					),
				),
			},
		},
	})
}

// newProposerAddrBz returns a random proposer address in bytes.
func newProposerAddrBz(t *testing.T) []byte {
	bech32 := sample.ConsAddress()
	addr, err := cosmostypes.ConsAddressFromBech32(bech32)
	require.NoError(t, err)

	return addr.Bytes()
}
