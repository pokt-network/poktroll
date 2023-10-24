package channel

import (
	"context"

	"pocket/pkg/observable"
)

// Map transforms the given observable by applying the given transformFn to each
// notification received from the observable. If the transformFn returns a skip
// bool of true, the notification is skipped and not emitted to the resulting
// observable.
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
