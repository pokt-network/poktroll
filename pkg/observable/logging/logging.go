package logging

import (
	"context"
	"log"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// LogErrors operates on an observable of errors. It logs all errors received
// from the observable.
func LogErrors(ctx context.Context, errs observable.Observable[error]) {
	channel.ForEach(ctx, errs, forEachErrorLogError)
}

// forEachErrorLogError is a ForEachFn that logs the given error.
func forEachErrorLogError(_ context.Context, err error) {
	log.Print(err)
}
