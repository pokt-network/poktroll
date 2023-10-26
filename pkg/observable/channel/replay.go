package channel

import (
	"context"
	"log"
	"sync"
	"time"

	"pocket/pkg/observable"
)

const replayNotificationTimeout = 1 * time.Second

var _ observable.ReplayObservable[any] = (*replayObservable[any])(nil)

type replayObservable[V any] struct {
	//*channelObservable[V]
	observableInternals[V]
	// replayBufferSize is the number of notifications to buffer so that they
	// can be replayed to new observers.
	replayBufferSize int
	// replayBufferMu protects replayBuffer from concurrent access/updates.
	replayBufferMu sync.RWMutex
	// replayBuffer is the buffer of notifications into which new notifications
	// will be pushed and which will be sent to new subscribers before any new
	// notifications are sent.
	replayBuffer []V
}

// NewReplayObservable returns a new ReplayObservable with the given replay buffer
// replayBufferSize and the corresponding publish channel to notify it of new values.
func NewReplayObservable[V any](
	ctx context.Context,
	replayBufferSize int,
) (observable.ReplayObservable[V], chan<- V) {
	obsvbl, publishCh := NewObservable[V]()
	return Replay[V](ctx, replayBufferSize, obsvbl), publishCh
}

// Replay returns an observable which replays the last replayBufferSize number of
// values published to the source observable to new observers, before publishing
// new values.
func Replay[V any](
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
//
//	duration before returning the accumulated notifications.
//
// If n is greater than the replay buffer size, the entire replay buffer is returned.
func (ro *replayObservable[V]) Last(ctx context.Context, n int) []V {
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

	// Replay all buffered replayBuffer to the observer channel buffer before
	// any new values have an opportunity to send on observerCh (i.e. appending
	// observer to ro.observers).
	//
	// TODO_IMPROVE: this assumes that the observer channel buffer is large enough
	// to hold all replay (buffered) replayBuffer.
	for _, notification := range ro.replayBuffer {
		observer.notify(notification)
	}

	// must (write) lock observersMu so that we can safely append to the observers list
	ro.observersMu.Lock()
	defer ro.observersMu.Unlock()

	// Explicitly append the observer to the observers list after replaying the
	// values in replayBuffer so that replayed notifications aren't re-added to it.
	ro.observers = append(ro.observers, observer)

	// caller can rely on context cancellation or call UnsubscribeAll() to unsubscribe
	// active observers
	if ctx != nil {
		// asynchronously wait for the context to be done and then unsubscribe
		// this observer.
		go goUnsubscribeOnDone[V](ctx, observer)
	}
	return observer
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
