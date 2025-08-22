package tokenomics

import (
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

var _ depinject.OnePerModuleType = AppModule{}

func init() {
	appconfig.Register(
		&types.Module{},
		appconfig.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	StoreService store.KVStoreService
	Cdc          codec.Codec
	Config       *types.Module
	Logger       log.Logger

	AccountKeeper      types.AccountKeeper
	BankKeeper         types.BankKeeper
	ApplicationKeeper  types.ApplicationKeeper
	SupplierKeeper     types.SupplierKeeper
	ProofKeeper        types.ProofKeeper
	SharedKeeper       types.SharedKeeper
	SessionKeeper      types.SessionKeeper
	ServiceKeeper      types.ServiceKeeper
	StakingKeeper      *stakingkeeper.Keeper
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

	// DEV_NOTE: The token logic modules are provided as arguments to the keeper mainly
	// to satisfy testing requirements (see: x/tokenomics/token_logic_modules_test.go).

	tokenLogicModules := tlm.NewDefaultTokenLogicModules()

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
		in.StakingKeeper,

		tokenLogicModules,
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
