package polyzero

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

type zerologULogger struct {
	// NB: Default (0) is Debug.
	level zerolog.Level
	zerolog.Logger
}

// TODO_IN_THIS_COMMIT: how to configure level?
func NewLogger(
	opts ...polylog.LoggerOption,
) polylog.Logger {
	ze := &zerologULogger{
		// Default to global  zerolog logger; stderr with timestamp.
		Logger: log.Logger,
	}

	ze.level = zerolog.DebugLevel

	for _, opt := range opts {
		opt(ze)
	}

	return ze
}

func (ze *zerologULogger) Debug() polylog.Event {
	return newEvent(ze.Logger.Debug())
}

func (ze *zerologULogger) Info() polylog.Event {
	return newEvent(ze.Logger.Info())
}

func (ze *zerologULogger) Warn() polylog.Event {
	return newEvent(ze.Logger.Warn())
}

func (ze *zerologULogger) Error() polylog.Event {
	return newEvent(ze.Logger.Error())
}
