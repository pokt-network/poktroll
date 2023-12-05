package polyzero

import (
	"io"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// WithOutput returns an option function that configures the output writer for zerolog.
func WithOutput(output io.Writer) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		zl := logger.(*zerologLogger).Logger
		logger.(*zerologLogger).Logger = zl.Output(output)
	}
}

// WithLevel returns an option function that configures the logger level for zerolog.
func WithLevel(level zerolog.Level) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		zl := logger.(*zerologLogger).Logger
		logger.(*zerologLogger).Logger = zl.Level(level)
	}
}

// WithTimestampKey returns an option function which configures the logger to
// use the given key when `polylog.Event#Timestamp()` is called.
func WithTimestampKey(key string) polylog.LoggerOption {
	return func(_ polylog.Logger) {
		zerolog.TimestampFieldName = key
	}
}

// WithErrKey returns an option function which configures the logger to use the
// given key when `polylog.Event#Err()` is called.
func WithErrKey(key string) polylog.LoggerOption {
	return func(_ polylog.Logger) {
		zerolog.ErrorFieldName = key
	}
}

// WithSetupFn takes function which receives the underlying zerolog logger pointer
// and returns an options function that calls it, passing the zerolog logger.
//
// TODO_TEST/TODO_COMMUNITY: add test coverage and example usage around this method.
func WithSetupFn(fn func(logger *zerolog.Logger)) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		fn(&logger.(*zerologLogger).Logger)
	}
}
