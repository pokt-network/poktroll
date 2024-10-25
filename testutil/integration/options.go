package integration

import (
	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
)

// IntegrationAppConfig is a configuration struct for an integration App. Its fields
// are intended to be set/updated by IntegrationAppOptionFn functions which are passed
// during integration App construction.
type IntegrationAppConfig struct {
	// InitChainerModuleFns are called for each module during the integration App's
	// InitChainer function.
	InitChainerModuleFns []InitChainerModuleFn
	TLMProcessors        []tlm.TokenLogicModuleProcessor
}

// IntegrationAppOptionFn is a function that receives and has the opportunity to
// modify the IntegrationAppConfig. It is intended to be passed during integration
// App construction to modify the behavior of the integration App.
type IntegrationAppOptionFn func(*IntegrationAppConfig)

// InitChainerModuleFn is a function that is called for each module during the
// integration App's InitChainer function.
type InitChainerModuleFn func(cosmostypes.Context, codec.Codec, appmodule.AppModule)

// WithInitChainerModuleFn returns an IntegrationAppOptionFn that appends the given
// InitChainerModuleFn to the IntegrationAppConfig's InitChainerModuleFns.
func WithInitChainerModuleFn(fn InitChainerModuleFn) IntegrationAppOptionFn {
	return func(config *IntegrationAppConfig) {
		config.InitChainerModuleFns = append(config.InitChainerModuleFns, fn)
	}
}

// WithModuleGenesisState returns an IntegrationAppOptionFn that appends an
// InitChainerModuleFn to the IntegrationAppConfig's InitChainerModuleFns which
// initializes the module's genesis state with the given genesisState. T is expected
// to be the module's AppModule type.
func WithModuleGenesisState[T module.HasGenesis](genesisState cosmostypes.Msg) IntegrationAppOptionFn {
	return WithInitChainerModuleFn(
		NewInitChainerModuleGenesisStateOptionFn[T](genesisState),
	)
}

// NewInitChainerModuleGenesisStateOptionFn returns an InitChainerModuleFn that
// initializes the module's genesis state with the given genesisState. T is expected
// to be the module's AppModule type.
func NewInitChainerModuleGenesisStateOptionFn[T module.HasGenesis](genesisState cosmostypes.Msg) InitChainerModuleFn {
	return func(ctx cosmostypes.Context, cdc codec.Codec, mod appmodule.AppModule) {
		targetModule, isTypeT := mod.(T)

		// Bail if this isn't the module we're looking for. 👋
		if !isTypeT {
			return
		}

		genesisJSON := cdc.MustMarshalJSON(genesisState)
		targetModule.InitGenesis(ctx, cdc, genesisJSON)
	}
}

// WithTLMProcessors returns an IntegrationAppOptionFn that sets the given
// TLM processors on the IntegrationAppConfig.
func WithTLMProcessors(processors []tlm.TokenLogicModuleProcessor) IntegrationAppOptionFn {
	return func(config *IntegrationAppConfig) {
		config.TLMProcessors = processors
	}
}
