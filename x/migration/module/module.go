package migration

import (
	"context"
	"encoding/json"
	"fmt"

	// this line is used by starport scaffolding # 1

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	modulev1 "github.com/pokt-network/poktroll/api/pocket/migration/module"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

var (
	_ module.AppModuleBasic      = (*AppModule)(nil)
	_ module.AppModuleSimulation = (*AppModule)(nil)
	_ module.HasGenesis          = (*AppModule)(nil)
	_ module.HasInvariants       = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)

	_ appmodule.AppModule       = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker = (*AppModule)(nil)
	_ appmodule.HasEndBlocker   = (*AppModule)(nil)
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface that defines the
// independent methods a Cosmos SDK module needs to implement.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the name of the module as a string.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module, which is used
// to marshal and unmarshal structs to/from []byte in order to persist them in the module's KVStore.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for testing.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	keeper         keeper.Keeper
	accountKeeper  types.AccountKeeper
	bankKeeper     types.BankKeeper
	sharedKeeper   types.SharedKeeper
	appKeeper      types.ApplicationKeeper
	supplierKeeper types.SupplierKeeper
	authzKeeper    types.AuthzKeeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	sharedKeeper types.SharedKeeper,
	appKeeper types.ApplicationKeeper,
	supplierKeeper types.SupplierKeeper,
	authzKeeper types.AuthzKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		sharedKeeper:   sharedKeeper,
		appKeeper:      appKeeper,
		supplierKeeper: supplierKeeper,
		authzKeeper:    authzKeeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// RegisterInvariants registers the invariants of the module. If an invariant deviates from its predicted value, the InvariantRegistry triggers appropriate logic (most often the chain will be halted)
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block.
// The begin block implementation is optional.
func (am AppModule) BeginBlock(ctx context.Context) error {
	granteeAddrStr := "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" //NetworkAuthzGranteeAddress[cosmosTypes.UnwrapSDKContext(ctx).ChainID()]
	granterAddrStr := "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t" //keepers.MigrationKeeper.GetAuthority()

	granteeAddr := cosmosTypes.AccAddress(granteeAddrStr)
	granterAddr := cosmosTypes.AccAddress(granterAddrStr)

	authorizations, _ := am.authzKeeper.GetAuthorizations(ctx, granteeAddr, granterAddr)
	for _, auth := range authorizations {
		am.keeper.Logger().Info(fmt.Sprintf("OLSH000: Found authorization for %s from %s to %s", auth.MsgTypeURL(), granteeAddrStr, granterAddrStr))
	}

	var grants []authz.GrantAuthorization
	am.authzKeeper.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, grant authz.Grant) bool {
		grants = append(grants, authz.GrantAuthorization{
			Granter:       granter.String(),
			Grantee:       grantee.String(),
			Expiration:    grant.Expiration,
			Authorization: grant.Authorization,
		})
		return false
	})
	logger := am.keeper.Logger()
	for _, entry := range grants {
		logger.Info(fmt.Sprintf("OLSH000: Found grant for %s from %s to %s", entry.Authorization.TypeUrl, entry.Grantee, entry.Granter))
		logger.Info(fmt.Sprintf("Grantee equal: %t", entry.Grantee == granteeAddrStr))
		logger.Info(fmt.Sprintf("Granter equal: %t", entry.Granter == granterAddrStr))
	}

	logger.Info(fmt.Sprintf("OLSH - Number of authorizations: %d", len(authorizations)))
	logger.Info(fmt.Sprintf("OLSH - Number of grants: %d", len(grants)))

	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(_ context.Context) error {
	return nil
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// ----------------------------------------------------------------------------
// App Wiring Setup
// ----------------------------------------------------------------------------

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	StoreService store.KVStoreService
	Cdc          codec.Codec
	Config       *modulev1.Module
	Logger       log.Logger

	AccountKeeper     types.AccountKeeper
	BankKeeper        types.BankKeeper
	SharedKeeper      types.SharedKeeper
	ApplicationKeeper types.ApplicationKeeper
	SupplierKeeper    types.SupplierKeeper
	AuthzKeeper       types.AuthzKeeper
}

type ModuleOutputs struct {
	depinject.Out

	MigrationKeeper keeper.Keeper
	Module          appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := keeper.NewKeeper(
		in.Cdc,
		in.StoreService,
		in.Logger,
		authority.String(),
		in.AccountKeeper,
		in.BankKeeper,
		in.SharedKeeper,
		in.ApplicationKeeper,
		in.SupplierKeeper,
		in.AuthzKeeper,
	)
	m := NewAppModule(
		in.Cdc,
		k,
		in.AccountKeeper,
		in.BankKeeper,
		in.SharedKeeper,
		in.ApplicationKeeper,
		in.SupplierKeeper,
		in.AuthzKeeper,
	)

	return ModuleOutputs{MigrationKeeper: k, Module: m}
}
