package tokenomics

import (
	"context"
	"encoding/json"
	// this line is used by starport scaffolding # 1

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	tokenomicsmodule "github.com/pokt-network/poktroll/api/poktroll/tokenomics/module"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
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
	return tokenomicstypes.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module, which is used
// to marshal and unmarshal structs to/from []byte in order to persist them in the module's KVStore.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers a module's interface tokenomicstypes and their concrete implementations as proto.Message.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	tokenomicstypes.RegisterInterfaces(reg)
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for testing.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(tokenomicstypes.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState tokenomicstypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return tokenomicstypes.ErrTokenomicsUnmarshalInvalid.Wrapf("invalid genesis state: %v", err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := tokenomicstypes.RegisterQueryHandlerClient(context.Background(), mux, tokenomicstypes.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	tokenomicsKeeper keeper.Keeper
	accountKeeper    tokenomicstypes.AccountKeeper
	bankKeeper       tokenomicstypes.BankKeeper
	supplierKeeper   tokenomicstypes.SupplierKeeper
}

func NewAppModule(
	cdc codec.Codec,
	tokenomicsKeeper keeper.Keeper,
	accountKeeper tokenomicstypes.AccountKeeper,
	bankKeeper tokenomicstypes.BankKeeper,
	supplierKeeper tokenomicstypes.SupplierKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic:   NewAppModuleBasic(cdc),
		tokenomicsKeeper: tokenomicsKeeper,
		accountKeeper:    accountKeeper,
		bankKeeper:       bankKeeper,
		supplierKeeper:   supplierKeeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	tokenomicstypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.tokenomicsKeeper))
	tokenomicstypes.RegisterQueryServer(cfg.QueryServer(), am.tokenomicsKeeper)
}

// RegisterInvariants registers the invariants of the module. If an invariant deviates from its predicted value, the InvariantRegistry triggers appropriate logic (most often the chain will be halted)
func (am AppModule) RegisterInvariants(_ cosmostypes.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx cosmostypes.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState tokenomicstypes.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.tokenomicsKeeper, genState)
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx cosmostypes.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.tokenomicsKeeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block.
// The begin block implementation is optional.
func (am AppModule) BeginBlock(_ context.Context) error {
	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(goCtx context.Context) error {
	ctx := cosmostypes.UnwrapSDKContext(goCtx)
	return EndBlocker(ctx, am.tokenomicsKeeper)
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// ----------------------------------------------------------------------------
// App Wiring Setup
// ----------------------------------------------------------------------------

func init() {
	appmodule.Register(&tokenomicsmodule.Module{}, appmodule.Provide(ProvideModule))
}

type ModuleInputs struct {
	depinject.In

	StoreService store.KVStoreService
	Cdc          codec.Codec
	Config       *tokenomicsmodule.Module
	Logger       log.Logger

	AccountKeeper     tokenomicstypes.AccountKeeper
	BankKeeper        tokenomicstypes.BankKeeper
	ApplicationKeeper tokenomicstypes.ApplicationKeeper
	SupplierKeeper    tokenomicstypes.SupplierKeeper
	ProofKeeper       tokenomicstypes.ProofKeeper
	SharedKeeper      tokenomicstypes.SharedKeeper
	SessionKeeper     tokenomicstypes.SessionKeeper
	ServiceKeeper     tokenomicstypes.ServiceKeeper
}

type ModuleOutputs struct {
	depinject.Out

	TokenomicsKeeper keeper.Keeper
	Module           appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	// MintAllocationDAO proportion of minted tokens will be sent to this address
	// as a result of global mint TLM processing.
	// TODO_TECHDEBT: Promote this value to a tokenomics module parameter.
	daoRewardBech32 := tokenomicstypes.DaoRewardAddress
	if daoRewardBech32 == "" {
		panic(`dao/foundation reward address MUST be set; add a "-X github.com/pokt-network/poktroll/x/tokenomics/types.DaoRewardAddress" element to build.ldglags in the config.yml`)
	}
	tlmProcessors := tlm.NewDefaultProcessors(daoRewardBech32)

	k := keeper.NewKeeper(
		in.Cdc,
		in.StoreService,
		in.Logger,
		authority.String(),

		in.BankKeeper,
		in.AccountKeeper,
		in.ApplicationKeeper,
		in.SupplierKeeper,
		in.ProofKeeper,
		in.SharedKeeper,
		in.SessionKeeper,
		in.ServiceKeeper,

		tlmProcessors,
	)
	m := NewAppModule(
		in.Cdc,
		k,
		in.AccountKeeper,
		in.BankKeeper,
		in.SupplierKeeper,
	)

	return ModuleOutputs{TokenomicsKeeper: k, Module: m}
}
