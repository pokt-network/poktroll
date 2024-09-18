package integration

import (
	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var (
	RunUntilNextBlockOpts = RunOptions{
		WithAutomaticCommit(),
		WithAutomaticFinalizeBlock(),
	}
)

// RunConfig is the configuration for the testing integration app.
type RunConfig struct {
	AutomaticFinalizeBlock bool
	AutomaticCommit        bool
	ErrorAssertion         func(error)
}

// RunOption is a function that can be used to configure the integration app.
type RunOption func(*RunConfig)

// RunOptions is a list of RunOption. It implements the Append method for convenience.
type RunOptions []RunOption

func (runOpts RunOptions) Config() *RunConfig {
	cfg := &RunConfig{}

	for _, opt := range runOpts {
		opt(cfg)
	}

	return cfg
}

// Append returns a new RunOptions with the given RunOptions appended.
func (runOpts RunOptions) Append(newRunOpts ...RunOption) RunOptions {
	return append(runOpts, newRunOpts...)
}

// WithAutomaticFinalizeBlock calls ABCI finalize block.
func WithAutomaticFinalizeBlock() RunOption {
	return func(cfg *RunConfig) {
		cfg.AutomaticFinalizeBlock = true
	}
}

// WithAutomaticCommit enables automatic commit.
// This means that the integration app will automatically commit the state after each msg.
func WithAutomaticCommit() RunOption {
	return func(cfg *RunConfig) {
		cfg.AutomaticCommit = true
	}
}

// WithErrorAssertion registers an error assertion function which is called when
// RunMsg() encounters an error.
func WithErrorAssertion(errAssertFn func(error)) RunOption {
	return func(cfg *RunConfig) {
		cfg.ErrorAssertion = errAssertFn
	}
}

// TODO_IN_THIS_COMMIT: godoc...
type IntegrationAppConfig struct {
	InitChainerModuleFns []InitChainerModuleFn
}

// TODO_IN_THIS_COMMIT: godoc...
type IntegrationAppOption func(*IntegrationAppConfig)

// TODO_IN_THIS_COMMIT: godoc...
type InitChainerModuleFn func(cosmostypes.Context, codec.Codec, appmodule.AppModule)

// TODO_IN_THIS_COMMIT: godoc...
func WithModuleGenesisState[T module.HasGenesis](genesisState cosmostypes.Msg) IntegrationAppOption {
	return func(config *IntegrationAppConfig) {
		config.InitChainerModuleFns = append(
			config.InitChainerModuleFns,
			NewInitChainerModuleGenesisStateOptionFn[T](genesisState),
		)
	}
}

// TODO_IN_THIS_COMMIT: godoc...
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
