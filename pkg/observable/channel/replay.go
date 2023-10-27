package channel

import (
	"context"
	"log"
	"sync"
	"time"

	"pocket/pkg/observable"
)

// replayPartialBufferTimeout is the duration to wait for the replay buffer to
// accumulate at least 1 value before returning the accumulated values.
// TODO_CONSIDERATION: perhaps this should be parameterized.
const replayPartialBufferTimeout = 100 * time.Millisecond

var _ observable.ReplayObservable[any] = (*replayObservable[any])(nil)

type replayObservable[V any] struct {
	// embed observerManager to encapsulate concurrent-safe read/write access to
	// observers. This also allows higher-level objects to wrap this observable
	// without knowing its specific type by asserting that it implements the
	// observerManager interface.
	observerManager[V]
	// replayBufferSize is the number of notifications to buffer so that they
	// can be replayed to new observers.
	replayBufferSize int
	// replayBufferMu protects replayBuffer from concurrent access/updates.
	replayBufferMu sync.RWMutex
	// replayBuffer holds the last relayBufferSize number of notifications received
	// by this observable. This buffer is replayed to new observers, on subscribing,
	// prior to any new notifications being propagated.
	replayBuffer []V
}

// NewReplayObservable returns a new ReplayObservable with the given replay buffer
// replayBufferSize and the corresponding publish channel to notify it of new values.
func NewReplayObservable[V any](
	ctx context.Context,
	replayBufferSize int,
) (observable.ReplayObservable[V], chan<- V) {
	obsvbl, publishCh := NewObservable[V]()
	return ToReplayObservable[V](ctx, replayBufferSize, obsvbl), publishCh
}

// ToReplayObservable returns an observable which replays the last replayBufferSize
// number of values published to the source observable to new observers, before
// publishing new values.
// It panics if srcObservable does not implement the observerManager interface.
// It should only be used with a srcObservable which contains channelObservers
// (i.e. channelObservable or similar).
func ToReplayObservable[V any](
	ctx context.Context,
	replayBufferSize int,
	srcObsvbl observable.Observable[V],
) observable.ReplayObservable[V] {
	// Assert that the source observable implements the observerMngr required
	// to embed and wrap it.
	observerMngr := srcObsvbl.(observerManager[V])

	replayObsvbl := &replayObservable[V]{
		observerManager:  observerMngr,
		replayBufferSize: replayBufferSize,
		replayBuffer:     make([]V, 0, replayBufferSize),
	}

	srcObserver := srcObsvbl.Subscribe(ctx)
	go replayObsvbl.goBufferReplayNotifications(srcObserver)

	return replayObsvbl
}

// Last synchronously returns the last n values from the replay buffer. It blocks
// until at least 1 notification has been accumulated, then waits replayPartialBufferTimeout
// duration before returning all notifications accumulated notifications by that time.
// If the replay buffer contains at least n notifications, this function will only
// block as long as it takes to accumulate and return them.
// If n is greater than the replay buffer size, the entire replay buffer is returned.
func (ro *replayObservable[V]) Last(ctx context.Context, n int) []V {
	// Use a temporary observer to accumulate replay values.
	// Subscribe will always start with the replay buffer, so we can safely
	// leverage it here for syncrhonization (i.e. blocking until at least 1
	// notification has been accumulated). This also eliminates the need for
	// locking and/or copying the replay buffer.
	tempObserver := ro.Subscribe(ctx)
	defer tempObserver.Unsubscribe()

	// If n is greater than the replay buffer size, return the entire replay buffer.
	if n > ro.replayBufferSize {
		n = ro.replayBufferSize
		log.Printf(
			"WARN: requested replay buffer size %d is greater than replay buffer capacity %d; returning entire replay buffer",
			n, cap(ro.replayBuffer),
		)
	}

	// accumulateReplayValues works concurrently and returns a context and cancellation
	// function for signaling completion.
	return accumulateReplayValues(tempObserver, n)
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (ro *replayObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	ro.replayBufferMu.RLock()
	defer ro.replayBufferMu.RUnlock()

	observer := NewObserver[V](ctx, ro.observerManager.remove)

	// Replay all buffered replayBuffer to the observer channel buffer before
	// any new values have an opportunity to send on observerCh (i.e. appending
	// observer to ro.observers).
	//
	// TODO_IMPROVE: this assumes that the observer channel buffer is large enough
	// to hold all replay (buffered) notifications.
	for _, notification := range ro.replayBuffer {
		observer.notify(notification)
	}

	ro.observerManager.add(observer)

	// caller can rely on context cancellation or call UnsubscribeAll() to unsubscribe
	// active observers
	if ctx != nil {
		// asynchronously wait for the context to be done and then unsubscribe
		// this observer.
		go ro.observerManager.goUnsubscribeOnDone(ctx, observer)
	}

	return observer
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (ro *replayObservable[V]) UnsubscribeAll() {
	ro.observerManager.removeAll()
}

// goBufferReplayNotifications buffers the last n notifications from a source
// observer. It is intended to be run in a goroutine.
func (ro *replayObservable[V]) goBufferReplayNotifications(srcObserver observable.Observer[V]) {
	for notification := range srcObserver.Ch() {
		ro.replayBufferMu.Lock()
		// Add the notification to the buffer.
		if len(ro.replayBuffer) < ro.replayBufferSize {
			ro.replayBuffer = append(ro.replayBuffer, notification)
		} else {
			// buffer full, make room for the new notification by removing the
			// oldest notification.
			ro.replayBuffer = append(ro.replayBuffer[1:], notification)
		}
		ro.replayBufferMu.Unlock()
	}
}

// accumulateReplayValues synchronously (but concurrently) accumulates n values
// from the observer channel into the slice pointed to by accValues and then returns
// said slice. It cancels the context either when n values have been accumulated
// or when at least 1 value has been accumulated and replayPartialBufferTimeout
// has elapsed.
func accumulateReplayValues[V any](observer observable.Observer[V], n int) []V {
	var (
		// accValuesMu protects accValues from concurrent access.
		accValuesMu sync.Mutex
		// Accumulate replay values in a new slice to avoid (read) locking replayBufferMu.
		accValues = new([]V)
		// Cancelling the context will cause the loop in the goroutine to exit.
		ctx, cancel = context.WithCancel(context.Background())
	)

	// Concurrently accumulate n values from the observer channel.
	go func() {
		// Defer cancelling the context and unlocking accValuesMu. The function
		// assumes that the mutex is locked when it gets execution control back
		// from the loop.
		defer func() {
			cancel()
			accValuesMu.Unlock()
		}()
		for {
			// Lock the mutex to read accValues here and potentially write in
			// the first case branch in the select below.
			accValuesMu.Lock()

			// The context was cancelled since the last iteration.
			if ctx.Err() != nil {
				return
			}

			// We've accumulated n values.
			if len(*accValues) >= n {
				return
			}

			// Receive from the observer's channel if we can, otherwise let
			// the loop run.
			select {
			// Receiving from the observer channel blocks if replayBuffer is empty.
			case value, ok := <-observer.Ch():
				// tempObserver was closed concurrently.
				if !ok {
					return
				}

				// Update the accumulated values pointed to by accValues.
				*accValues = append(*accValues, value)
			default:
				// If we can't receive from the observer channel immediately,
				// let the loop run.
			}

			// Unlock accValuesMu so that the select below gets a chance to check
			// the length of *accValues to decide whether to cancel, and it can
			// be relocked at the top of the loop as it must be locked when the
			// loop exits.
			accValuesMu.Unlock()
			// Wait a tick before continuing the loop.
			time.Sleep(time.Millisecond)
		}
	}()

	// Wait for N values to be accumulated or timeout. When timing out, if we
	// have at least 1 value, we can return it. Otherwise, we need to wait for
	// the next value to be published (i.e. continue the loop).
	for {
		select {
		case <-ctx.Done():
			return *accValues
		case <-time.After(replayPartialBufferTimeout):
			accValuesMu.Lock()
			if len(*accValues) > 1 {
				cancel()
				return *accValues
			}
			accValuesMu.Unlock()
		}
	}
}
