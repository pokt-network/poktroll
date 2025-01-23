package polyzero

import (
	"context"
	"os"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// zerologLogger is a thin wrapper around a zerolog logger
// plus a map of fields for deduplication.
type zerologLogger struct {
	// NB: Default (0) is Debug.
	level  zerolog.Level
	fields map[string]any // store accumulated fields as a map
	zerolog.Logger
}

// NewLogger constructs a new zerolog-backed logger which conforms to the
// polylog.Logger interface. By default, the logger is configured to write to
// os.Stderr and log at the Debug level.
//
// TODO_IMPROVE/TODO_COMMUNITY: Add `NewProductionLogger`, `NewDevelopmentLogger`,
// and `NewExampleLogger` functions with reasonable defaults their respective
// environments; conceptually similar to the respective analogues in zap.
// See: https://pkg.go.dev/github.com/uber-go/zap#hdr-Configuring_Zap.
func NewLogger(
	opts ...polylog.LoggerOption,
) polylog.Logger {
	ze := &zerologLogger{
		level:  zerolog.DebugLevel,
		fields: make(map[string]any),
		Logger: zerolog.New(os.Stderr),
	}

	for _, opt := range opts {
		opt(ze)
	}

	return ze
}

// Debug starts a new message with debug level.
//
// You must call Msg on the returned event in order to send the event.
func (ze *zerologLogger) Debug() polylog.Event {
	return newEvent(ze.Logger.Debug())
}

// Info starts a new message with info level.
//
// You must call Msg, Msgf, or Send on the returned event in order to send the event.
func (ze *zerologLogger) Info() polylog.Event {
	return newEvent(ze.Logger.Info())
}

// Warn starts a new message with warn level.
//
// You must call Msg, Msgf, or Send on the returned event in order to send the event.
func (ze *zerologLogger) Warn() polylog.Event {
	return newEvent(ze.Logger.Warn())
}

// Error starts a new message with error level.
//
// You must call Msg, Msgf, or Send on the returned event in order to send the event.
func (ze *zerologLogger) Error() polylog.Event {
	return newEvent(ze.Logger.Error())
}

// With deduplicates the provided key-values by copying the parent's
// fields, overriding any keys in the parent's map, and returning
// a new logger instance. This avoids repeated keys in the final JSON.
func (ze *zerologLogger) With(keyVals ...any) polylog.Logger {
	newFields := make(map[string]any) // start with a fresh map

	// Merge new fields, overriding duplicates
	for i := 0; i < len(keyVals); i += 2 {
		if i+1 < len(keyVals) {
			if key, ok := keyVals[i].(string); ok {
				newFields[key] = keyVals[i+1]
			}
		}
	}

	// Create a new logger with the same level
	newLogger := ze.Logger.Level(ze.level)

	// Create a new context builder
	ctx := newLogger.With()

	// Add fields one by one to ensure proper deduplication
	for k, v := range newFields {
		ctx = ctx.Interface(k, v)
	}

	// Return the new logger instance
	return &zerologLogger{
		level:  ze.level,
		fields: newFields,
		Logger: ctx.Logger(),
	}
}

// copyMap is a helper to clone a map[string]any.
func copyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// WithLevel starts a new message with level.
//
// You must call Msg, Msgf, or Send on the returned event in order to send the event.
func (ze *zerologLogger) WithLevel(level polylog.Level) polylog.Event {
	return newEvent(ze.Logger.WithLevel(zerolog.Level(level.Int())))
}

// WithContext returns a copy of ctx with the receiver logger attached.
//   - The Logger attached to the provided Context (if any) will not be effected.
//   - If the receiver's log level is Disabled, it will only be attached to the returned
//     Context if the provided Context has a previously attached Logger.
//   - If the provided Context has no attached Logger, a Disabled Logger
//     will not be attached.
//
// TODO_TEST/TODO_COMMUNITY: add support for #UpdateContext() and update this
// godoc to inlude example usage.
// See: https://pkg.go.dev/github.com/rs/zerolog#Logger.WithContext.
//
// TODO_TEST/TODO_COMMUNITY: add coverage for `polyzero.Logger#WithContext()`.
func (ze *zerologLogger) WithContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, polylog.PolylogCtxKey, ze)
	ctx = ze.Logger.WithContext(ctx)
	return ctx
}

// Write implements io.Writer. This is useful to set as a writer for the
// standard library log.
func (ze *zerologLogger) Write(p []byte) (n int, err error) {
	return ze.Logger.Write(p)
}
