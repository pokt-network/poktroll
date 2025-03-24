package channel

import (
	"context"

	"github.com/pokt-network/pocket/pkg/observable"
)

type MapFn[S, D any] func(ctx context.Context, src S) (dst D, skip bool)
type ForEachFn[V any] func(ctx context.Context, src V)

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

	go goMapTransformNotification(
		ctx,
		srcObserver,
		transformFn,
		func(dstNotification D) {
			dstProducer <- dstNotification
		},
		dstProducer,
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

	go goMapTransformNotification(
		ctx,
		srcObserver,
		transformFn,
		func(dstNotifications []D) {
			for _, dstNotification := range dstNotifications {
				dstPublishCh <- dstNotification
			}
		},
		dstPublishCh,
	)

	return dstObservable
}

// MapReplay transforms the given observable by applying the given transformFn to
// each notification received from the observable. If the transformFn returns a
// skip bool of true, the notification is skipped and not emitted to the resulting
// observable.
// The resulting observable will receive the last replayBufferCap
// number of values published to the source observable before receiving new values.
func MapReplay[S, D any](
	ctx context.Context,
	replayBufferCap int,
	srcObservable observable.Observable[S],
	transformFn MapFn[S, D],
) observable.ReplayObservable[D] {
	dstObservable, dstProducer := NewReplayObservable[D](ctx, replayBufferCap)
	srcObserver := srcObservable.Subscribe(ctx)

	go goMapTransformNotification(
		ctx,
		srcObserver,
		transformFn,
		func(dstNotification D) {
			dstProducer <- dstNotification
		},
		dstProducer,
	)

	return dstObservable
}

// ForEach applies the given forEachFn to each notification received from the
// observable, similar to Map; however, ForEach does not publish to a destination
// observable. ForEach is useful for side effects and is a terminal observable
// operator.
func ForEach[V any](
	ctx context.Context,
	srcObservable observable.Observable[V],
	forEachFn ForEachFn[V],
) {
	Map(
		ctx, srcObservable,
		func(ctx context.Context, src V) (dst V, skip bool) {
			forEachFn(ctx, src)

			// No downstream observers; SHOULD always skip.
			return zeroValue[V](), true
		},
	)
}

// goMapTransformNotification transforms, optionally skips, and publishes
// notifications via the given publishFn.
func goMapTransformNotification[S, D, P any](
	ctx context.Context,
	srcObserver observable.Observer[S],
	transformFn MapFn[S, D],
	publishFn func(dstNotifications D),
	// dstProducerCh is created by the caller of goMapTransformNotification
	// which does not have knowledge as of when srcObserver.Ch() is closed.
	// It is passed to the goMapTransformNotification to close it when the
	// function is done.
	dstProducerCh chan<- P,
) {
	for srcNotification := range srcObserver.Ch() {
		dstNotifications, skip := transformFn(ctx, srcNotification)
		if skip {
			continue
		}

		publishFn(dstNotifications)
	}
	close(dstProducerCh)
}

// zeroValue is a generic helper which returns the zero value of the given type.
func zeroValue[T any]() (zero T) {
	return zero
}
