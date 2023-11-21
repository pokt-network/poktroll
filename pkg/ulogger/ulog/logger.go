package ulog

import (
	"github.com/pokt-network/poktroll/pkg/ulogger"
)

//var _ ulogger.UniversalLogger = (*stdLogULogger)(nil)

type stdLogULogger struct{}

func NewUniversalLogger() ulogger.UniversalLogger {
	return &stdLogULogger{}
}

func (st *stdLogULogger) Debug() ulogger.Event {
	return newEvent(ulogger.LevelDebug)
}

func (st *stdLogULogger) Info() ulogger.Event {
	return newEvent(ulogger.LevelInfo)
}

func (st *stdLogULogger) Warn() ulogger.Event {
	return newEvent(ulogger.LevelWarn)
}

func (st *stdLogULogger) Error() ulogger.Event {
	return newEvent(ulogger.LevelError)
}
