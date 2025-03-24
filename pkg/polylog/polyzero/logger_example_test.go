package polyzero_test

import (
	"os"

	"github.com/pokt-network/pocket/pkg/polylog/polyzero"
)

func ExampleNewLogger() {
	// Use whichever level you need.
	level := polyzero.InfoLevel
	// Specify the lowest level to log. I.e.: calls to level methods "lower"
	// than this will be ignored.
	levelOpt := polyzero.WithLevel(level)

	// Construct logger.
	// NB: adding WithOutput is optional; defaults to os.Stderr. It is needed
	// here to print to stdout for testable example purposes.
	logger := polyzero.NewLogger(levelOpt, polyzero.WithOutput(os.Stdout))

	// All level methods are always available, but will only log if the level
	// is enabled.
	logger.Debug().Msg("debug message - should not see me")
	logger.Info().Msgf("info message with %s", "formatting")
	logger.Warn().Str("warn", "message").Send()
	// NB: arg type MUST be either map[string]any OR []any.
	logger.Error().Fields(map[string]any{
		"error": "message",
	}).Send()

	// Output:
	// {"level":"info","message":"info message with formatting"}
	// {"level":"warn","warn":"message"}
	// {"level":"error","error":"message"}
}
