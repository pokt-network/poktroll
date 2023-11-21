package uzap

import (
	"io"

	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

func WithOutput(output io.Writer) ulogger.LoggerOption {
	return func(ul ulogger.UniversalLogger) {
		ul.(*zapULogger).writeSyncer = zapcore.AddSync(output)
	}
}

func WithLevel(level zapcore.Level) ulogger.LoggerOption {
	return func(ul ulogger.UniversalLogger) {
		ul.(*zapULogger).level = level
	}
}
