package logging

import (
	"context"
	"log"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

func LogErrors(ctx context.Context, errs observable.Observable[error]) {
	channel.ForEach(ctx, errs, forEachErrorLogError)
}

func forEachErrorLogError(_ context.Context, err error) {
	log.Print(err)
}
