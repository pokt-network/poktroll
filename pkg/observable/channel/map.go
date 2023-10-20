package channel

import (
	"context"

	"pocket/pkg/observable"
)

func Map[S, D any](
	ctx context.Context,
	srcObservable observable.Observable[S],
	// TODO_CONSIDERATION: if this were variadic, it could simplify serial transformations.
	transformFn func(src S) (dst D, skip bool),
) observable.Observable[D] {
	dstObservable, dstProducer := NewObservable[D]()
	srcObserver := srcObservable.Subscribe(ctx)

	go func() {
		for srcNotification := range srcObserver.Ch() {
			dstNotification, skip := transformFn(srcNotification)
			if skip {
				continue
			}

			dstProducer <- dstNotification
		}
	}()

	return dstObservable
}
