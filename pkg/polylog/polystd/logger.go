package polystd

import (
	"github.com/pokt-network/poktroll/pkg/polylog"
)

//var _ polylog.PolyLogger = (*stdLogULogger)(nil)

type stdLogULogger struct{}

func NewPolyLogger() polylog.PolyLogger {
	return &stdLogULogger{}
}

func (st *stdLogULogger) Debug() polylog.Event {
	return newEvent(polylog.LevelDebug)
}

func (st *stdLogULogger) Info() polylog.Event {
	return newEvent(polylog.LevelInfo)
}

func (st *stdLogULogger) Warn() polylog.Event {
	return newEvent(polylog.LevelWarn)
}

func (st *stdLogULogger) Error() polylog.Event {
	return newEvent(polylog.LevelError)
}
