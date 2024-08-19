package channel

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// TODO_DISCUSS: what should this be? should it be configurable? It seems to be most
// relevant in the context of the behavior of the observable when it has multiple
// observers which consume at different rates.
// defaultSubscribeBufferSize is the buffer size of a observable's publish channel.
const defaultPublishBufferSize = 50

var (
	_ observable.Observable[any] = (*channelObservable[any])(nil)
	_ observerManager[any]       = (*channelObservable[any])(nil)
)

// option is a function which receives and can modify the channelObservable state.
type option[V any] func(obs *channelObservable[V])

// channelObservable implements the observable.Observable interface and can be notified
// by sending on its corresponding publishCh channel.
type channelObservable[V any] struct {
	// embed observerManager to encapsulate concurrent-safe read/write access to
	// observers. This also allows higher-level objects to wrap this observable
	// without knowing its specific type by asserting that it implements the
	// observerManager interface.
	observerManager[V]
	// publishCh is an observable-wide channel that is used to receive values
	// which are subsequently fanned out to observers.
	publishCh chan V
}

// NewObservable creates a new observable which is notified when the publishCh
// channel receives a value.
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observable that publishes messages from 1 publishCh to N observers
	obs := &channelObservable[V]{
		observerManager: newObserverManager[V](),
	}

	for _, opt := range opts {
		opt(obs)
	}

	// If the caller does not provide a publishCh, create a new one using the
	// defaultPublishBuffer size and return it.
	if obs.publishCh == nil {
		obs.publishCh = make(chan V, defaultPublishBufferSize)
	}

	// start listening to the publishCh and emit values to observers
	go obs.goPublish()

	return obs, obs.publishCh
}

// WithPublisher returns an option function which sets the given publishCh of the
// resulting observable when passed to NewObservable().
func WithPublisher[V any](publishCh chan V) option[V] {
	return func(obs *channelObservable[V]) {
		obs.publishCh = publishCh
	}
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (obs *channelObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	if ctx == nil {
		ctx = context.Background()
	}

	// caller can cancel context or close the publish channel to unsubscribe active observers
	ctx, cancel := context.WithCancel(ctx)
	removeAndCancel := func(toRemove observable.Observer[V]) {
		obs.observerManager.remove(toRemove)
		cancel()
	}

	// Create a new observer and add it to the list of observers to be notified
	// when publishCh receives a new value.
	observer := NewObserver[V](ctx, removeAndCancel)
	obs.observerManager.add(observer)

	// asynchronously wait for the context to be done and then unsubscribe
	// this observer.
	go obs.observerManager.goUnsubscribeOnDone(ctx, observer)

	return observer
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (obs *channelObservable[V]) UnsubscribeAll() {
	obs.observerManager.removeAll()
}

// goPublish to the publishCh and notify observers when values are received.
// This function is blocking and should be run in a goroutine.
func (obs *channelObservable[V]) goPublish() {
	// TODO_INVESTIGATE: Relayminer: lots of goroutines wait here. At what point do we ever close this channel?
	for notification := range obs.publishCh {
		obs.observerManager.notifyAll(notification)
	}

	fmt.Println("DIMA_CHANNEL_CLOSED")
	debug.PrintStack()

	// Here we know that the publisher channel has been closed.
	// Unsubscribe all observers as they can no longer receive notifications.
	obs.observerManager.removeAll()
}
