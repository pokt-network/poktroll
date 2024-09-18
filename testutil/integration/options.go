package integration

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
