package polyzero

import (
	"io"

	"github.com/rs/zerolog"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

func WithOutput(output io.Writer) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		zl := zerolog.New(output)
		logger.(*zerologULogger).Logger = zl
	}
}

func WithLevel(level zerolog.Level) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		logger.(*zerologULogger).level = level
	}
}

func WithSetupFn(fn func(logger *zerolog.Logger)) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		fn(&logger.(*zerologULogger).Logger)
	}
}
