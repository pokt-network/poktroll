package polystd

import (
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ polylog.Logger = (*stdLogULogger)(nil)

type stdLogULogger struct{}

func NewPolyLogger() polylog.Logger {
	return &stdLogULogger{}
}

func (st *stdLogULogger) Debug() polylog.Event {
	return newEvent(DebugLevel)
}

func (st *stdLogULogger) Info() polylog.Event {
	return newEvent(InfoLevel)
}

func (st *stdLogULogger) Warn() polylog.Event {
	return newEvent(WarnLevel)
}

func (st *stdLogULogger) Error() polylog.Event {
	return newEvent(ErrorLevel)
}
