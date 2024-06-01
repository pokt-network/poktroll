package integration

import (
	"context"
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
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
)

const appName = "integration-app"

// App is a test application that can be used to test the integration of modules.
type App struct {
	*baseapp.BaseApp

	ctx           sdk.Context
	logger        log.Logger
	moduleManager module.Manager
	queryHelper   *baseapp.QueryServiceTestHelper
}

func NewIntegrationApp(
	t *testing.T,
	sdkCtx sdk.Context,
	logger log.Logger,
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

	// if keys[consensusparamtypes.StoreKey] != nil {
	// 	// set baseApp param store
	// 	consensusParamsKeeper := consensusparamkeeper.NewKeeper(
	// 		cdc,
	// 		runtime.NewKVStoreService(storeKeys[banktypes.StoreKey]),
	// 		logger,
	// 		autho

	// 		log.NewNopLogger(), runtime.EnvWithQueryRouterService(grpcRouter), runtime.EnvWithMsgRouterService(msgRouter)), authtypes.NewModuleAddress("gov").String())
	// 	bApp.SetParamStore(consensusParamsKeeper.ParamsStore)
	// 	consensusparamtypes.RegisterQueryServer(grpcRouter, consensusParamsKeeper)

	// 	params := cmttypes.ConsensusParamsFromProto(*simtestutil.DefaultConsensusParams) // This fills up missing param sections
	// 	err := consensusParamsKeeper.ParamsStore.Set(sdkCtx, params.ToProto())
	// 	if err != nil {
	// 		panic(fmt.Errorf("failed to set consensus params: %w", err))
	// 	}

	// 	if err := bApp.LoadLatestVersion(); err != nil {
	// 		panic(fmt.Errorf("failed to load application version from store: %w", err))
	// 	}

	// 	if _, err := bApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName, ConsensusParams: simtestutil.DefaultConsensusParams}); err != nil {
	// 		panic(fmt.Errorf("failed to initialize application: %w", err))
	// 	}
	// } else {

	err := bApp.LoadLatestVersion()
	require.NoError(t, err, "failed to load latest version")

	_, err = bApp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: appName})
	require.NoError(t, err, "failed to initialize chain")
	// }

	_, err = bApp.Commit()
	require.NoError(t, err, "failed to commit")

	cometHeader := cmtproto.Header{ChainID: appName}
	ctx := sdkCtx.WithBlockHeader(cometHeader).WithIsCheckTx(true)

	return &App{
		BaseApp:       bApp,
		logger:        logger,
		ctx:           ctx,
		moduleManager: *moduleManager,
		queryHelper:   baseapp.NewQueryServerTestHelper(ctx, interfaceRegistry),
	}
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

// Context returns the application context. It can be unwrapped to a sdk.Context,
// with the sdk.UnwrapSDKContext function.
func (app *App) Context() context.Context {
	return app.ctx
}

// QueryHelper returns the application query helper.
// It can be used when registering query services.
func (app *App) QueryHelper() *baseapp.QueryServiceTestHelper {
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
