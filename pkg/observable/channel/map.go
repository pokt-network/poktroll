package channel

import (
	"context"
	"runtime"
	"sync"

	"github.com/pokt-network/poktroll/pkg/observable"
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

// MapParallel behaves like Map but applies transformFn concurrently across
// numWorkers goroutines that all read from the same source observer.
//
// IMPORTANT: OUTPUT ORDER IS NOT PRESERVED. Workers race to consume and publish,
// so notifications may be emitted in a different order than received. Only use
// MapParallel where the downstream consumer is order-independent — e.g. inserting
// into a set or a sparse-merkle-sum-trie keyed by a content hash, where insertion
// is commutative. For ordered/serial semantics use Map.
//
// It is intended for CPU-bound, pure, per-item transforms (e.g. marshal + hash)
// that would otherwise bottleneck on Map's single consumer goroutine.
//
// numWorkers <= 0 falls back to runtime.GOMAXPROCS(0).
func MapParallel[S, D any](
	ctx context.Context,
	srcObservable observable.Observable[S],
	transformFn MapFn[S, D],
	numWorkers int,
	opts ...option[D],
) observable.Observable[D] {
	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}

	dstObservable, dstProducer := NewObservable[D](opts...)
	srcObserver := srcObservable.Subscribe(ctx)

	go func() {
		var wg sync.WaitGroup
		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func() {
				defer wg.Done()
				// Each worker drains the shared source channel; channel receive is
				// safe across goroutines and delivers every value to exactly one worker.
				for srcNotification := range srcObserver.Ch() {
					dstNotification, skip := transformFn(ctx, srcNotification)
					if skip {
						continue
					}
					// Concurrent sends to dstProducer are safe (channel sends are
					// goroutine-safe); the destination observable fans them out.
					dstProducer <- dstNotification
				}
			}()
		}
		// Close the destination producer only after ALL workers have observed the
		// source channel close and drained — closing earlier would drop in-flight work.
		wg.Wait()
		close(dstProducer)
	}()

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
