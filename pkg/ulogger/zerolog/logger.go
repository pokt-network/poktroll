package zerolog

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

type zerologULogger struct {
	zerolog.Logger
}

func NewUniversalLogger(
	opts ...ulogger.LoggerOption,
) ulogger.UniversalLogger {
	ze := &zerologULogger{
		// Default to global  zerolog logger; stderr with timestamp.
		Logger: log.Logger,
	}

	for _, opt := range opts {
		opt(ze)
	}

	return ze
}

func (ze *zerologULogger) Debug() ulogger.Event {
	return newEvent(ze.Logger.Debug())
}

func (ze *zerologULogger) Info() ulogger.Event {
	return newEvent(ze.Logger.Info())
}

func (ze *zerologULogger) Warn() ulogger.Event {
	return newEvent(ze.Logger.Warn())
}

func (ze *zerologULogger) Error() ulogger.Event {
	return newEvent(ze.Logger.Error())
}
