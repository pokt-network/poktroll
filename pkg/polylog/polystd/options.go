package polystd

import (
	"io"
	"log"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

func WithOutput(output io.Writer) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		log.SetOutput(output)
	}
}

func WithLevel(level Level) polylog.LoggerOption {
	return func(logger polylog.Logger) {
		logger.(*stdLogLogger).level = level
	}
}
