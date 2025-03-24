package logging

import (
	"context"

	"github.com/pokt-network/pocket/pkg/observable"
	"github.com/pokt-network/pocket/pkg/observable/channel"
	"github.com/pokt-network/pocket/pkg/polylog"
)

// LogErrors operates on an observable of errors. It logs all errors received
// from the observable.
func LogErrors(ctx context.Context, errs observable.Observable[error]) {
	channel.ForEach(ctx, errs, forEachErrorLogError)
}

// forEachErrorLogError is a ForEachFn that logs the given error.
func forEachErrorLogError(ctx context.Context, err error) {
	logger := polylog.Ctx(ctx)
	// Logging the error and flushing (i.e. sending) the log message to stdout
	logger.Error().Err(err).Send()
}
