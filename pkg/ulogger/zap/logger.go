package zap

import (
	"go.uber.org/zap"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

var _ ulogger.UniversalLogger = (*zapULogger)(nil)

type zapULogger struct {
	logger *zap.Logger
}

func (za *zapULogger) Debug() ulogger.Event {
	//return (log.Debug())
	return nil
}

func (za *zapULogger) Info() ulogger.Event {
	return nil
}

func (za *zapULogger) Warn() ulogger.Event {
	return nil
}

func (za *zapULogger) Error() ulogger.Event {
	return nil
}

// TODO_IN_THIS_COMMIT: Implement Fatal & Panic ?
