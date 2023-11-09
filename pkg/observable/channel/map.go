package channel

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/observable"
)

type MapFn[S, D any] func(ctx context.Context, src S) (dst D, skip bool)

// Map transforms the given observable by applying the given transformFn to each
// notification received from the observable. If the transformFn returns a skip
// bool of true, the notification is skipped and not emitted to the resulting
// observable.
func Map[S, D any](
	ctx context.Context,
	srcObservable observable.Observable[S],
	transformFn MapFn[S, D],
) observable.Observable[D] {
	dstObservable, dstProducer := NewObservable[D]()
	srcObserver := srcObservable.Subscribe(ctx)

	mapInternal[S, D](
		ctx,
		srcObserver,
		transformFn,
		func(dstNotification D) {
			dstProducer <- dstNotification
		},
	)

	return dstObservable
}

// MapExpand transforms the given observable by applying the given transformFn to
// each notification received from the observable, similar to Map; however, the
// transformFn returns a slice of output notifications for each input notification.
func MapExpand[S, D any](
	ctx context.Context,
	srcObservable observable.Observable[S],
	transformFn MapFn[S, []D],
) observable.Observable[D] {
	dstObservable, dstPublishCh := NewObservable[D]()
	srcObserver := srcObservable.Subscribe(ctx)

	mapInternal[S, []D](
		ctx,
		srcObserver,
		transformFn,
		func(dstNotifications []D) {
			for _, dstNotification := range dstNotifications {
				dstPublishCh <- dstNotification
			}
		},
	)

	return dstObservable
}

func mapInternal[S, D any](
	ctx context.Context,
	srcObserver observable.Observer[S],
	transformFn MapFn[S, D],
	publishFn func(dstNotifications D),
) {
	go func() {
		for srcNotification := range srcObserver.Ch() {
			dstNotifications, skip := transformFn(ctx, srcNotification)
			if skip {
				continue
			}

			publishFn(dstNotifications)
		}
	}()

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
