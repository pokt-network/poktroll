package channel

import (
	"context"
	"fmt"
	"sync"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var (
	_ observable.ReplayObservable[any] = (*replayObservable[any])(nil)
	_ observable.Observable[any]       = (*replayObservable[any])(nil)
)

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
	opts ...option[V],
) (observable.ReplayObservable[V], chan<- V) {
	obsvbl, publishCh := NewObservable[V](opts...)
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

	srcObserver := replayObsvbl.Subscribe(ctx)
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
	ro.replayBufferMu.RLock()
	defer ro.replayBufferMu.RUnlock()
	logger := polylog.Ctx(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If n is greater than the replay buffer size, return the entire replay buffer.
	if n > ro.replayBufferSize {
		n = ro.replayBufferSize
		logger.Warn().
			Int("requested_replay_buffer_size", n).
			Int("replay_buffer_capacity", cap(ro.replayBuffer)).
			Msg("requested replay buffer size is greater than replay buffer capacity; returning entire replay buffer")
	}

	// accumulateReplayValues works concurrently and returns a context and cancellation
	// function for signaling completion.
	var accValues []V
	copySize := n
	if len(ro.replayBuffer) < n {
		copySize = len(ro.replayBuffer)
	}
	logger.Debug().Msgf("replayBuffer %v", ro.replayBuffer)
	for i := len(ro.replayBuffer) - copySize; i < len(ro.replayBuffer); i++ {
		accValues = append(accValues, ro.replayBuffer[i])
	}

	if len(accValues) < n {
		sub := ro.Subscribe(ctx)
		for value := range sub.Ch() {
			logger.Debug().
				Int("len", len(accValues)).
				Msgf("replayObservable.Last: received value: %v", value)
			accValues = append(accValues, value)
			if len(accValues) >= n {
				break
			}
		}
	}

	return accValues
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (ro *replayObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	//fmt.Println("Subscribing before locking")
	//ro.replayBufferMu.RLock()
	//fmt.Println("Subscribing locked")
	//defer ro.replayBufferMu.RUnlock()

	if ctx == nil {
		ctx = context.Background()
	}

	// caller can cancel context or close the publish channel to unsubscribe active observers
	ctx, cancel := context.WithCancel(ctx)
	removeAndCancel := func(toRemove observable.Observer[V]) {
		ro.observerManager.remove(toRemove)
		cancel()
	}

	observer := NewObserver[V](ctx, removeAndCancel)

	// Replay all buffered replayBuffer to the observer channel buffer before
	// any new values have an opportunity to send on observerCh (i.e. appending
	// observer to ro.observers).
	//
	// TODO_IMPROVE: this assumes that the observer channel buffer is large enough
	// to hold all replay (buffered) notifications.
	for _, notification := range ro.replayBuffer {
		observer.notify(notification)
	}

	fmt.Println("before adding observer")
	ro.observerManager.add(observer)
	fmt.Println("afeter adding observer")

	// asynchronously wait for the context to be done and then unsubscribe
	// this observer.
	go ro.observerManager.goUnsubscribeOnDone(ctx, observer)

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
		fmt.Println("buffer got value, locking")
		ro.replayBufferMu.Lock()
		fmt.Println("buffer locked")
		// Add the notification to the buffer.
		if len(ro.replayBuffer) < ro.replayBufferSize {
			ro.replayBuffer = append(ro.replayBuffer, notification)
		} else {
			// buffer full, make room for the new notification by removing the
			// oldest notification.
			ro.replayBuffer = append(ro.replayBuffer[1:], notification)
		}
		fmt.Println("filling buffer")
		ro.replayBufferMu.Unlock()
	}
}
