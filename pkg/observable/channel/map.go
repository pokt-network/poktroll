package channel

import (
	"context"

	"pocket/pkg/observable"
)

type MapFn[S, D any] func(src S) (dst D, skip bool)

// Map transforms the given observable by applying the given transformFn to each
// notification received from the observable. If the transformFn returns a skip
// bool of true, the notification is skipped and not emitted to the resulting
// observable.
func Map[S, D any](
	ctx context.Context,
	srcObservable observable.Observable[S],
	// TODO_CONSIDERATION: if this were variadic, it could simplify serial transformations.
	transformFn MapFn[S, D],
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

// MapReplay transforms the given observable by applying the given transformFn to
// each notification received from the observable. If the transformFn returns a
// skip bool of true, the notification is skipped and not emitted to the resulting
// observable.
// The resulting observable will receive the last replayBufferSize
// number of values published to the source observable before receiving new values.
func MapReplay[S, D any](
	ctx context.Context,
	replayBufferSize int,
	srcObservable observable.Observable[S],
	// TODO_CONSIDERATION: if this were variadic, it could simplify serial transformations.
	transformFn func(src S) (dst D, skip bool),
) observable.ReplayObservable[D] {
	dstObservable, dstProducer := NewReplayObservable[D](ctx, replayBufferSize)
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
