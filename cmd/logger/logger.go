package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	// LogLevel holds the logging level from --log-level flag.
	// Set by commands that call PreRunESetup().
	LogLevel string

	// LogOutput holds the log output destination from --log-output flag.
	// Set by commands that call PreRunESetup().
	LogOutput string

	// Logger is the configured global logger instance.
	// Configured in PreRunESetup() based on LogLevel and LogOutput values.
	Logger polylog.Logger
)

const (
	unknownLevel = "???"

	outputDiscard = "discard"
	outputStdout  = "stdout"
	outputStderr  = "stderr"
)

// PreRunESetup configures the global Logger based on LogLevel and LogOutput values.
// Should be called from (or by) a Cobra command's PreRunE function.
// Features:
// • Thread-safe, non-blocking logging via diode wrapper
// • Supports stdout, stderr, discard, or file output
// • Sets logger on command context
//
// TODO_CONSIDERATION: Apply this pattern to all CLI commands.
func PreRunESetup(cmd *cobra.Command, _ []string) error {
	var (
		logWriter io.Writer
		err       error
	)

	switch LogOutput {
	case flags.DefaultLogOutput, outputStdout:
		logWriter = os.Stdout
	case outputStderr:
		logWriter = os.Stderr
	case outputDiscard:
		logWriter = io.Discard
	default:
		logWriter, err = os.OpenFile(LogOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
	}

	// Wrap the writer in a thread-safe, lock-free, non-blocking io.Writer.
	logWriter = diode.NewWriter(logWriter, 1000, 10*time.Millisecond, func(int) {})
	logLevel := polyzero.ParseLevel(LogLevel)
	Logger = polyzero.NewLogger(
		polyzero.WithLevel(logLevel),
		polyzero.WithSetupFn(NewSetupConsoleWriter(logWriter)),
	)

	// Set the logger on the context and update the command's context.
	loggerCtx := Logger.WithContext(cmd.Context())
	cmd.SetContext(loggerCtx)

	return nil
}

// NewSetupConsoleWriter returns a polylog setup function which wraps the underlying
// zerolog logger in a ConsoleWriter, for prettier output. The console writer is
// configured to exclude the timestamp field, and to exclude the log level output
// for the info level to reduce output verbosity.
// See: https://github.com/rs/zerolog/?tab=readme-ov-file#pretty-logging.
func NewSetupConsoleWriter(logWriter io.Writer) func(zlog *zerolog.Logger) {
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
