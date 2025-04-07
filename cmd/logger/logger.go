package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	// LogLevel is a global variable that is intended to hold the value of the
	// "--log-level" flag when a command which has called PreRunESetup() is executed.
	LogLevel string

	// LogOutput is a global variable that is intended to hold the value of the
	// "--log-output" flag when a command which has called PreRunESetup() is executed.
	LogOutput string

	// Logger is a global variable that holds the logger which is configured according
	// to the values of the LogLevel and LogOutput global variables in PreRunESetup().
	Logger polylog.Logger
)

const unknownLevel = "???"

// PreRunESetup sets up the global cmd logger (Logger) for use in any subcommand.
// This function is intended to be passed as (or called by) a `PreRunE` function
// of a Cobra command.
//
// TODO_CONSIDERATION: Apply this pattern to all CLI commands.
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
		polyzero.WithSetupFn(newSetupConsoleWriter(logWriter)),
	)

	return nil
}

// newSetupConsoleWriter returns a polylog setup function which wraps the underlying
// zerolog logger in a ConsoleWriter, for prettier output. The console writer is
// configured to exclude the timestamp field, and to exclude the log level output
// for the info level to reduce output verbosity.
func newSetupConsoleWriter(logWriter io.Writer) func(zlog *zerolog.Logger) {
	return func(zlog *zerolog.Logger) {
		*zlog = zlog.Output(zerolog.ConsoleWriter{
			Out:          logWriter,
			PartsExclude: []string{zerolog.TimestampFieldName},
			FormatLevel:  newLevelFormatter(false, zerolog.InfoLevel),
		})
	}
}

// newLevelFormatter returns a zerolog Formatter function that is used to conditionally
// format the log level part of the output. It is derived from zerolog's default level
// formatter, which is not exported (at the time of writing). The original behavior is
// extended to support exclusion of the level part for a variadic number of levels,
// given by excludeLevelParts.
func newLevelFormatter(noColor bool, excludeLevelParts ...zerolog.Level) zerolog.Formatter {
	return func(i interface{}) string {
		if ll, ok := i.(string); ok {
			level, _ := zerolog.ParseLevel(ll)
			fl, ok := zerolog.FormattedLevels[level]
			if ok {
				for _, excludeLevel := range excludeLevelParts {
					if level == excludeLevel {
						return ""
					}
				}

				return colorize(fl, zerolog.LevelColors[level], noColor)
			}
			return stripLevel(ll)
		}
		if i == nil {
			return unknownLevel
		}
		return stripLevel(fmt.Sprintf("%s", i))
	}
}

// colorize returns the string s wrapped in ANSI code c, unless disabled is true or c is 0.
// It is derived from zerolog's colorize function, which is not exported (at the time of writing).
func colorize(s any, c int, disabled bool) string {
	e := os.Getenv("NO_COLOR")
	if e != "" || c == 0 {
		disabled = true
	}

	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

// stripLevel returns the log level part of the given string, or "???" if the string
// is empty or does not contain a valid log level.
// It is derived from zerolog's stripLevel function, which is not exported (at the time of writing).
func stripLevel(ll string) string {
	if len(ll) == 0 {
		return unknownLevel
	}
	if len(ll) > 3 {
		ll = ll[:3]
	}
	return strings.ToUpper(ll)
}
