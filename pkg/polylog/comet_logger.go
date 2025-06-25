package polylog

import (
	"github.com/cometbft/cometbft/libs/log"
)

// polylogCometLoggerWrapper is an adapter that implements the CometBFT log.Logger
// interface using the application's polylog.Logger.
//
// This wrapper allows our application logger to be used with CometBFT components
// that expect a specific logging interface.
type polylogCometLoggerWrapper struct {
	Logger // Embed the polylog Logger
}

// Info implements the CometBFT log.Logger interface's Info method.
// It forwards log messages to the polylog Logger's Info level with the given key-value pairs.
func (l *polylogCometLoggerWrapper) Info(msg string, keyvals ...interface{}) {
	l.Logger.With(keyvals...).Info().Msg(msg)
}

// Debug implements the CometBFT log.Logger interface's Debug method.
// It forwards log messages to the polylog Logger's Debug level with the given key-value pairs.
func (l *polylogCometLoggerWrapper) Debug(msg string, keyvals ...interface{}) {
	l.Logger.With(keyvals...).Debug().Msg(msg)
}

// Error implements the CometBFT log.Logger interface's Error method.
// It forwards log messages to the polylog Logger's Error level with the given key-value pairs.
func (l *polylogCometLoggerWrapper) Error(msg string, keyvals ...interface{}) {
	l.Logger.With(keyvals...).Error().Msg(msg)
}

// With implements the CometBFT log.Logger interface's With method.
// It creates a new logger with the given key-value pairs added to the logging context.
func (l *polylogCometLoggerWrapper) With(keyvals ...interface{}) log.Logger {
	// Since both interfaces use the same format for key-value pairs,
	// we can simply pass the arguments directly to the underlying Logger
	return &polylogCometLoggerWrapper{Logger: l.Logger.With(keyvals...)}
}

// ToCometLogger adapts a polylog.Logger instance to a CometBFT log.Logger interface:
// - Allows application's logger to work with CometBFT components
// - Bridges between different logging interface requirements
// - Maintains consistent logging patterns across the application
// - Satisfies CometBFT's specific logging interface needs
func ToCometLogger(logger Logger) log.Logger {
	return &polylogCometLoggerWrapper{Logger: logger}
}
