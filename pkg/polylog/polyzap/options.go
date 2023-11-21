package polyzap

import (
	"io"

	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

func WithOutput(output io.Writer) polylog.LoggerOption {
	return func(ul polylog.PolyLogger) {
		ul.(*zapULogger).writeSyncer = zapcore.AddSync(output)
	}
}

func WithLevel(level zapcore.Level) polylog.LoggerOption {
	return func(ul polylog.PolyLogger) {
		ul.(*zapULogger).level = level
	}
}
