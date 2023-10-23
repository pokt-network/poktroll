package channel

import (
	"context"
	"sync"
	"time"

	"pocket/pkg/observable"
)

const replayNotificationTimeout = 1 * time.Second

var _ observable.Observable[any] = &replayObservable[any]{}

type replayObservable[V any] struct {
	*channelObservable[V]
	size              int
	notificationsMu   sync.RWMutex
	notifications     []V
	replayObserversMu sync.RWMutex
	replayObservers   []observable.Observer[V]
}

// Replay returns an observable which replays the last n values published to the
// source observable to new observers, before publishing new values.
func Replay[V any](
	ctx context.Context, n int,
	srcObsvbl observable.Observable[V],
) observable.Observable[V] {
	// TODO_HACK/TODO_IMPROVE: more effort is required to make a generic replay
	// observable; however, as we only have the one observable package (channel),
	// and aren't anticipating need another, we can get away with this for now.
	chanObsvbl, ok := srcObsvbl.(*channelObservable[V])
	if !ok {
		panic("Replay only supports channelObservable")
	}

	replayObsvbl := &replayObservable[V]{
		channelObservable: chanObsvbl,
		size:              n,
		notifications:     make([]V, 0, n),
	}

	srcObserver := srcObsvbl.Subscribe(ctx)
	go replayObsvbl.goBufferReplayNotifications(srcObserver)

	return replayObsvbl
}

// Next synchronously returns the next value from the observable. This will always
// return the first value in the replay buffer, if it exists.
func (ro *replayObservable[V]) Next(ctx context.Context) V {
	tempObserver := ro.Subscribe(ctx)
	defer tempObserver.Unsubscribe()

	val := <-tempObserver.Ch()
	return val
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (ro *replayObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	ro.notificationsMu.RLock()
	defer ro.notificationsMu.RUnlock()

	observer := NewObserver[V](ctx, ro.onUnsubscribe)

	// Replay all buffered notifications to the observer channel buffer before
	// any new values have an opportunity to send on observerCh (i.e. appending
	// observer to ro.observers).
	//
	// TODO_IMPROVE: this assumes that the observer channel buffer is large enough
	// to hold all replay (buffered) notifications.
	for _, notification := range ro.notifications {
		observer.notify(notification)
	}

	// must (write) lock observersMu so that we can safely append to the observers list
	ro.observersMu.Lock()
	defer ro.observersMu.Unlock()

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

// goBufferReplayNotifications buffers the last n notifications from a source
// observer. It is intended to be run in a goroutine.
func (ro *replayObservable[V]) goBufferReplayNotifications(srcObserver observable.Observer[V]) {
	for notification := range srcObserver.Ch() {
		ro.notificationsMu.Lock()
		// Add the notification to the buffer.
		if len(ro.notifications) < ro.size {
			ro.notifications = append(ro.notifications, notification)
		} else {
			// buffer full, make room for the new notification by removing the
			// oldest notification.
			ro.notifications = append(ro.notifications[1:], notification)
		}
		ro.notificationsMu.Unlock()
	}
}
