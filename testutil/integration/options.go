package integration

import (
	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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

// TODO_IN_THIS_COMMIT: godoc
type RunOptions []RunOption

// TODO_IN_THIS_COMMIT: godoc
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

// TODO_IN_THIS_COMMIT: move...
type IntegrationAppConfig struct {
	initChainerModuleFns []IntegrationAppOption
}

// TODO_IN_THIS_COMMIT: move...
type IntegrationAppOption func(
	cosmostypes.Context,
	codec.Codec,
	appmodule.AppModule,
) (isGenesisImported bool)
