package zerolog

import (
	"io"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/ulogger"
)

func WithOutput(output io.Writer) ulogger.LoggerOption {
	return func(logger ulogger.UniversalLogger) {
		zl := zerolog.New(output)
		logger.(*zerologULogger).Logger = zl
	}
}

func WithSetupFn(fn func(logger *zerolog.Logger)) ulogger.LoggerOption {
	return func(logger ulogger.UniversalLogger) {
		fn(&logger.(*zerologULogger).Logger)
	}
}
