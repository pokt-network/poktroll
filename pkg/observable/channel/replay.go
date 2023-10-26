package channel

import (
	"context"
	"log"
	"sync"
	"time"

	"pocket/pkg/observable"
)

// TODO_CONSIDERATION: perhaps this should be parameterized.
const replayPartialBufferTimeout = 100 * time.Millisecond

var _ observable.ReplayObservable[any] = (*replayObservable[any])(nil)

type replayObservable[V any] struct {
	//*channelObservable[V]
	observableInternals[V]
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
// It panics if srcObservable does not implement the observableInternals interface.
func ToReplayObservable[V any](
	ctx context.Context,
	replayBufferSize int,
	srcObsvbl observable.Observable[V],
) observable.ReplayObservable[V] {
	// Assert that the source observable implements the internals required to
	// embed and wrap it.
	internals := srcObsvbl.(observableInternals[V])

	replayObsvbl := &replayObservable[V]{
		observableInternals: internals,
		replayBufferSize:    replayBufferSize,
		replayBuffer:        make([]V, 0, replayBufferSize),
	}

	srcObserver := srcObsvbl.Subscribe(ctx)
	go replayObsvbl.goBufferReplayNotifications(srcObserver)

	return replayObsvbl
}

// Last synchronously returns the last n values from the replay buffer. It blocks
// until at least 1 notification has been accumulated, then waits replayPartialBufferTimeout
// duration before returning the accumulated notifications.
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

	// Accumulate replay values in a new slice to avoid (read)
	// locking replayBufferMu.
	var values []V
	gotNValues := accumulateNValues(ctx, tempObserver, n, &values)

	// Wait for N values to be accumulated or timeout. When timing out, if we
	// have at least 1 value, we can return it. Otherwise, we need to wait for
	// the next value to be published (i.e. continue the loop).
	for {
		select {
		case <-gotNValues:
			return values
		case <-time.After(replayPartialBufferTimeout):
			if len(values) > 1 {
				return values
			}
		}
	}
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (ro *replayObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	ro.replayBufferMu.RLock()
	defer ro.replayBufferMu.RUnlock()

	observer := NewObserver[V](ctx, ro.onUnsubscribe)

	// ToReplayObservable all buffered replayBuffer to the observer channel buffer before
	// any new values have an opportunity to send on observerCh (i.e. appending
	// observer to ro.observers).
	//
	// TODO_IMPROVE: this assumes that the observer channel buffer is large enough
	// to hold all replay (buffered) replayBuffer.
	for _, notification := range ro.replayBuffer {
		observer.notify(notification)
	}

	ro.addObserver(observer)

	return observer
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (ro *replayObservable[V]) UnsubscribeAll() {
	ro.unsubscribeAll()
}

// goBufferReplayNotifications buffers the last n replayBuffer from a source
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

func accumulateNValues[V any](
	ctx context.Context,
	observer observable.Observer[V],
	n int, accValues *[]V,
) (done chan struct{}) {
	done = make(chan struct{}, 1)
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			if len(*accValues) >= n {
				done <- struct{}{}
				return
			}

			select {
			// Receiving from the observer channel blocks if replayBuffer is empty.
			case value, ok := <-observer.Ch():
				// tempObserver was closed concurrently.
				if !ok {
					return
				}

				*accValues = append(*accValues, value)
			default:
			}

			time.Sleep(time.Millisecond)
		}
	}()
	return done
}
