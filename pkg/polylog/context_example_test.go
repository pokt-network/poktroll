package polylog_test

import (
	"context"
	"os"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func ExampleCtx() {
	// Use whichever zerolog level you need.
	level := polyzero.InfoLevel

	// Specify the lowest level to log. I.e.: calls to level methods "lower"
	// than this will be ignored.
	levelOpt := polyzero.WithLevel(level)

	// Construct a context, this is typically received as an argument.
	ctx := context.Background()

	// Construct expectedLogger.
	// NB: adding WithOutput is optional; defaults to os.Stderr. It is needed
	// here to print to stdout for testable example purposes.
	expectedLogger := polyzero.NewLogger(levelOpt, polyzero.WithOutput(os.Stdout))

	// Add fields to the expectedLogger's context so that we can identify it in the output.
	expectedLogger = expectedLogger.With("label", "my_test_logger")

	// Associate the expectedLogger with the context and update the context reference.
	ctx = expectedLogger.WithContext(ctx)

	// Retrieve the logger from the context.
	retrievedLogger := polylog.Ctx(ctx)

	// Log and check that the output matches our expectations.
	retrievedLogger.Info().Msg("info message")

	// Output:
	// {"level":"info","label":"my_test_logger","message":"info message"}
}
