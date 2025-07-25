package app

import (
	// this line is used by starport scaffolding # stargate/app/moduleImport
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	testdata_pulsar "github.com/cosmos/cosmos-sdk/testutil/testdata/testpb"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/docs"
	"github.com/pokt-network/poktroll/telemetry"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const (
	AccountAddressPrefix = "pokt"
	Name                 = "pocket"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	claimMorseAcctMsgTypeUrl     = sdk.MsgTypeURL(&migrationtypes.MsgClaimMorseAccount{})
	claimMorseAppMsgTypeUrl      = sdk.MsgTypeURL(&migrationtypes.MsgClaimMorseApplication{})
	claimMorseSupplierMsgTypeUrl = sdk.MsgTypeURL(&migrationtypes.MsgClaimMorseSupplier{})
)

var (
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*runtime.App
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry
	Keepers           keepers.Keepers

	// simulation manager
	sm *module.SimulationManager

	// this line is used by starport scaffolding # stargate/app/keeperDeclaration
	// MUST_READ_DEV_NOTE: Ignite CLI adds keepers here when scaffolding new modules.
	// MUST_READ_DEV_ACTION_ITEM: Please move the created keeper to the `keepers` package.
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}

// getGovProposalHandlers return the chain proposal handlers.
func getGovProposalHandlers() []govclient.ProposalHandler {
	var govProposalHandlers []govclient.ProposalHandler
	// this line is used by starport scaffolding # stargate/app/govProposalHandlers

	govProposalHandlers = append(govProposalHandlers,
		paramsclient.ProposalHandler,
		// this line is used by starport scaffolding # stargate/app/govProposalHandler
	)

	return govProposalHandlers
}

// AppConfig returns the default app config.
func AppConfig() depinject.Config {
	return depinject.Configs(
		appConfig,
		// Loads the ao config from a YAML file.
		// appconfig.LoadYAML(AppConfigYAML),
		depinject.Supply(
			// supply custom module basics
			map[string]module.AppModuleBasic{
				genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
				govtypes.ModuleName:     gov.NewAppModuleBasic(getGovProposalHandlers()),
				// this line is used by starport scaffolding # stargate/appConfig/moduleBasic
			},
		),
	)
}

// New returns a reference to an initialized App.
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) (*App, error) {
	var (
		app = &App{
			Keepers: keepers.Keepers{},
		}
		appBuilder *runtime.AppBuilder

		// merge the AppConfig and other configuration in one config
		deps = depinject.Configs(
			AppConfig(),
			depinject.Supply(
				// Supply the application options
				appOpts,
				// Supply with IBC keeper getter for the IBC modules with App Wiring.
				// The IBC Keeper cannot be passed because it has not been initiated yet.
				// Passing the getter, the app IBC Keeper will always be accessible.
				// This needs to be removed after IBC supports App Wiring.
				app.GetIBCKeeper,
				app.GetCapabilityScopedKeeper,
				// Supply the logger
				logger,

				// ADVANCED CONFIGURATION
				//
				// AUTH
				//
				// For providing a custom function required in auth to generate custom account types
				// add it below. By default the auth module uses simulation.RandomGenesisAccounts.
				//
				// authtypes.RandomGenesisAccountsFn(simulation.RandomGenesisAccounts),
				//
				// For providing a custom a base account type add it below.
				// By default the auth module uses authtypes.ProtoBaseAccount().
				//
				// func() sdk.AccountI { return authtypes.ProtoBaseAccount() },
				//
				// For providing a different address codec, add it below.
				// By default the auth module uses a Bech32 address codec,
				// with the prefix defined in the auth module configuration.
				//
				// func() address.Codec { return <- custom address codec type -> }

				//
				// STAKING
				//
				// For providing a different validator and consensus address codec, add it below.
				// By default the staking module uses the bech32 prefix provided in the auth config,
				// and appends "valoper" and "valcons" for validator and consensus addresses respectively.
				// When providing a custom address codec in auth, custom address codecs must be provided here as well.
				//
				// func() runtime.ValidatorAddressCodec { return <- custom validator address codec type -> }
				// func() runtime.ConsensusAddressCodec { return <- custom consensus address codec type -> }

				//
				// MINT
				//

				// For providing a custom inflation function for x/mint add here your
				// custom function that implements the minttypes.InflationCalculationFn
				// interface.
			),
		)
	)

	if err := depinject.Inject(deps,
		&appBuilder,
		&app.appCodec,
		&app.legacyAmino,
		&app.txConfig,
		&app.interfaceRegistry,
		&app.Keepers.AccountKeeper,
		&app.Keepers.BankKeeper,
		&app.Keepers.StakingKeeper,
		&app.Keepers.SlashingKeeper,
		&app.Keepers.MintKeeper,
		&app.Keepers.DistrKeeper,
		&app.Keepers.GovKeeper,
		&app.Keepers.CrisisKeeper,
		&app.Keepers.UpgradeKeeper,
		&app.Keepers.ParamsKeeper,
		&app.Keepers.AuthzKeeper,
		&app.Keepers.EvidenceKeeper,
		&app.Keepers.FeeGrantKeeper,
		&app.Keepers.GroupKeeper,
		&app.Keepers.ConsensusParamsKeeper,
		&app.Keepers.CircuitBreakerKeeper,
		&app.Keepers.ServiceKeeper,
		&app.Keepers.GatewayKeeper,
		&app.Keepers.ApplicationKeeper,
		&app.Keepers.SupplierKeeper,
		&app.Keepers.SessionKeeper,
		&app.Keepers.ProofKeeper,
		&app.Keepers.TokenomicsKeeper,
		&app.Keepers.SharedKeeper,
		&app.Keepers.MigrationKeeper,
		// this line is used by starport scaffolding # stargate/app/keeperDefinition
		// MUST_READ_DEV_NOTE: Ignite CLI adds keepers here when scaffolding new modules.
		// MUST_READ_DEV_ACTION_ITEM: Please move the created keeper to the `keepers` package.
	); err != nil {
		panic(err)
	}

	// Below we could construct and set an application specific mempool and
	// ABCI 1.0 PrepareProposal and ProcessProposal handlers. These defaults are
	// already set in the SDK's BaseApp, this shows an example of how to override
	// them.
	//
	// Example:
	//
	// app.App = appBuilder.Build(...)
	// nonceMempool := mempool.NewSenderNonceMempool()
	// abciPropHandler := NewDefaultProposalHandler(nonceMempool, app.App.BaseApp)
	//
	// app.App.BaseApp.SetMempool(nonceMempool)
	// app.App.BaseApp.SetPrepareProposal(abciPropHandler.PrepareProposalHandler())
	// app.App.BaseApp.SetProcessProposal(abciPropHandler.ProcessProposalHandler())
	//
	// Alternatively, you can construct BaseApp options, append those to
	// baseAppOptions and pass them to the appBuilder.
	//
	// Example:
	//
	// prepareOpt = func(app *baseapp.BaseApp) {
	// 	abciPropHandler := baseapp.NewDefaultProposalHandler(nonceMempool, app)
	// 	app.SetPrepareProposal(abciPropHandler.PrepareProposalHandler())
	// }
	// baseAppOptions = append(baseAppOptions, prepareOpt)
	//
	// create and set vote extension handler
	// voteExtOp := func(bApp *baseapp.BaseApp) {
	// 	voteExtHandler := NewVoteExtensionHandler()
	// 	voteExtHandler.SetHandlers(bApp)
	// }

	// Setup the application with block metrics that hook into the ABCI handlers.
	// TODO_TECHDEBT: Use a flag to enable/disable block metrics.
	baseAppOptions = append(baseAppOptions, telemetry.InitBlockMetrics)

	app.App = appBuilder.Build(db, traceStore, baseAppOptions...)

	// Set a custom ante handler to waive minimum gas/fees for transactions
	// IF the migration module's `waive_morse_claim_gas_fees` param is true.
	// The ante handler waives fees for txs which contain ONLY morse claim
	// messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and
	// MsgClaimMorseSupplier), and is signed by a single secp256k1 signer.
	app.SetAnteHandler(newMorseClaimGasFeesWaiverAnteHandlerFn(app))

	// Register legacy modules
	app.registerIBCModules()

	// register streaming services
	if err := app.RegisterStreamingServices(appOpts, app.kvStoreKeys()); err != nil {
		return nil, err
	}

	/****  Module Options ****/

	//nolint:staticcheck // SA1019 TODO_TECHDEBT(#1276): remove deprecated code.
	app.ModuleManager.RegisterInvariants(app.Keepers.CrisisKeeper)

	// add test gRPC service for testing gRPC queries in isolation
	testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.Keepers.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	// Custom InitChainer to ensure ICA host port binding.
	//
	// Why this is necessary:
	// - The default InitChainer does NOT bind the "icahost" port
	// - Binding this port is required to allow other chains (controllers) to open ICA channels with pocket (host)
	// - Without this, controller chains will get "port not found" errors during handshake
	// - Even if ICA host is disabled via `host_enabled = false`, binding the port is harmless
	// - Port binding is a runtime state operation and MUST happen after capability initialization
	app.SetInitChainer(func(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
		// Parse genesis state (this replaces the default InitChainer logic)
		var genesisState map[string]json.RawMessage
		if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis state: %w", err)
		}

		// Initialize all modules from genesis
		res, err := app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
		if err != nil {
			return nil, fmt.Errorf("failed to init genesis: %w", err)
		}

		// Bind the ICA host port (if not already bound)
		if !app.Keepers.IBCKeeper.PortKeeper.IsBound(ctx, icahosttypes.SubModuleName) {
			// This allows remote controller chains to create ICA channels
			_ = app.Keepers.IBCKeeper.PortKeeper.BindPort(ctx, icahosttypes.SubModuleName)
		}

		return res, nil
	})

	if err := app.setUpgrades(); err != nil {
		return nil, err
	}

	if err := app.Load(loadLatest); err != nil {
		return nil, err
	}

	// Set up pocket telemetry using `app.toml` configuration options (in addition to cosmos-sdk telemetry config).
	if err := telemetry.New(appOpts); err != nil {
		return nil, err
	}

	return app, nil
}

// LegacyAmino returns App's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
// DEV_NOTE: Do not delete this.
// It is needed to comply with the ignite CLI; https://github.com/ignite/cli/issues/4697
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// TxConfig returns App's transaction config.
// DEV_NOTE: Do not delete this.
// It is needed to comply with the ignite CLI; https://github.com/ignite/cli/issues/4697
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	kvStoreKey, ok := app.UnsafeFindStoreKey(storeKey).(*storetypes.KVStoreKey)
	if !ok {
		return nil
	}
	return kvStoreKey
}

// GetMemKey returns the MemoryStoreKey for the provided store key.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	key, ok := app.UnsafeFindStoreKey(storeKey).(*storetypes.MemoryStoreKey)
	if !ok {
		return nil
	}

	return key
}

// kvStoreKeys returns all the kv store keys registered inside App.
func (app *App) kvStoreKeys() map[string]*storetypes.KVStoreKey {
	keys := make(map[string]*storetypes.KVStoreKey)
	for _, k := range app.GetStoreKeys() {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			keys[kv.Name()] = kv
		}
	}

	return keys
}

// GetSubspace returns a param subspace for a given module name.
func (app *App) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.Keepers.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface.
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	app.App.RegisterAPIRoutes(apiSvr, apiConfig)
	// register swagger API in app.go so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}

	// register app's OpenAPI routes.
	docs.RegisterOpenAPIService(Name, apiSvr.Router)
}

// GetIBCKeeper returns the IBC keeper.
func (app *App) GetIBCKeeper() *ibckeeper.Keeper {
	return app.Keepers.IBCKeeper
}

// GetCapabilityScopedKeeper returns the capability scoped keeper.
func (app *App) GetCapabilityScopedKeeper(moduleName string) capabilitykeeper.ScopedKeeper {
	return app.Keepers.CapabilityKeeper.ScopeToModule(moduleName)
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for _, perms := range moduleAccPerms {
		dup[perms.Account] = perms.Permissions
	}
	return dup
}

// BlockedAddresses returns all the app's blocked account addresses.
// It is returned as a map for easy lookup, but is in essence a set.
func BlockedAddresses() map[string]bool {
	blockedAddressSet := make(map[string]bool)
	if len(blockAccAddrs) > 0 {
		for _, addr := range blockAccAddrs {
			blockedAddressSet[addr] = true
		}
	} else {
		for addr := range GetMaccPerms() {
			blockedAddressSet[addr] = true
		}
	}
	return blockedAddressSet
}
