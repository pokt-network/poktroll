package testpolylog

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func NewLoggerWithCtx(
	ctx context.Context,
	level polylog.Level,
) (polylog.Logger, context.Context) {
	levelOpt := polyzero.WithLevel(level)
	logger := polyzero.NewLogger(levelOpt)
	ctx = logger.WithContext(ctx)

	return logger, ctx
}
