package polyzap

import (
	"io"

	"go.uber.org/zap/zapcore"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

func WithOutput(output io.Writer) polylog.LoggerOption {
	return func(ul polylog.Logger) {
		ul.(*zapLogger).writeSyncer = zapcore.AddSync(output)
	}
}

func WithLevel(level Level) polylog.LoggerOption {
	return func(zl polylog.Logger) {
		// TODO_IN_THIS_COMMIT: consider comparing level.String() & zapcore.Level.String().
		zl.(*zapLogger).level = zapcore.Level(level.Int())
	}
}
