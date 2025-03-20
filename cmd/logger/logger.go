package logger

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	// LogLevel is a global variable that is intended to hold the value of the
	// "--log-level" flag when a command which has called PreRunESetup is executed.
	LogLevel string

	// LogOutput is a global variable that is intended to hold the value of the
	// "--log-output" flag when a command which has called PreRunESetup() is executed.
	LogOutput string

	// Logger is a global variable that holds the logger which is configured according
	// to the values of the LogLevel and LogOutput global variables in PreRunESetup().
	Logger polylog.Logger
)

// PreRunESetup sets up the global cmd logger (Logger) for use in any subcommand.
// This function is intended to be passed as (or called by) a `PreRunE` function
// of a Cobra command.
func PreRunESetup(_ *cobra.Command, _ []string) error {
	var (
		logWriter io.Writer
		err       error
	)

	logLevel := polyzero.ParseLevel(LogLevel)
	if LogOutput == flags.DefaultLogOutput {
		logWriter = os.Stdout
	} else {
		logWriter, err = os.Open(LogOutput)
		if err != nil {
			return err
		}
	}

	Logger = polyzero.NewLogger(
		polyzero.WithLevel(logLevel),
		polyzero.WithOutput(logWriter),
	).With("cmd", "migrate")

	return nil
}
