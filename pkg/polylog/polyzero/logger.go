package polyzero

import (
	"context"
	"os"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// zerologLogger is a thin wrapper around a zerolog logger which implements
// the polylog.Logger interface.
type zerologLogger struct {
	// NB: Default (0) is Debug.
	level zerolog.Level
	zerolog.Logger
}

// NewLogger constructs a new zerolog-backed logger which conforms to the
// polylog.Logger interface. By default, the logger is configured to write to
// os.Stderr and log at the Debug level.
//
// TODO_IMPROVE/TODO_COMMUNITY: Add `NewProductionLogger`, `NewDevelopmentLogger`,
// and `NewExampleLogger` functions with reasonable defaults the their respective
// environments; conceptually similar to the respective analogues in zap.
// See: https://pkg.go.dev/github.com/uber-go/zap#hdr-Configuring_Zap.
func NewLogger(
	opts ...polylog.LoggerOption,
) polylog.Logger {
	ze := &zerologLogger{
		level:  zerolog.DebugLevel,
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

// With creates a child logger with the fields constructed from keyVals added
// to its context.
func (ze *zerologLogger) With(keyVals ...any) polylog.Logger {
	return &zerologLogger{
		level:  ze.level,
		Logger: ze.Logger.With().Fields(keyVals).Logger(),
	}
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
