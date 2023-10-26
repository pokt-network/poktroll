package channel

import (
	"context"
	"pocket/pkg/observable"
)

// TODO_DISCUSS: what should this be? should it be configurable? It seems to be most
// relevant in the context of the behavior of the observable when it has multiple
// observers which consume at different rates.
// defaultSubscribeBufferSize is the buffer size of a observable's publish channel.
const defaultPublishBufferSize = 50

var (
	_ observable.Observable[any] = (*channelObservable[any])(nil)
	_ observableInternals[any]   = (*channelObservable[any])(nil)
)

// option is a function which receives and can modify the channelObservable state.
type option[V any] func(obs *channelObservable[V])

// channelObservable implements the observable.Observable interface and can be notified
// by sending on its corresponding publishCh channel.
type channelObservable[V any] struct {
	//observableInternals[V]
	channelObservableInternals[V]
	// publishCh is an observable-wide channel that is used to receive values
	// which are subsequently fanned out to observers.
	publishCh chan V
}

// NewObservable creates a new observable which is notified when the publishCh
// channel receives a value.
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observable that publishes messages from 1 publishCh to N observers
	obs := &channelObservable[V]{
		channelObservableInternals: newObservableInternals[V](),
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
func (obsvbl *channelObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	// Create a new observer and add it to the list of observers to be notified
	// when publishCh receives a new value.
	observer := NewObserver[V](ctx, obsvbl.onUnsubscribe)
	obsvbl.addObserver(observer)

	// caller can rely on context cancellation or call UnsubscribeAll() to unsubscribe
	// active observers
	if ctx != nil {
		// asynchronously wait for the context to be done and then unsubscribe
		// this observer.
		go goUnsubscribeOnDone[V](ctx, observer)
	}
	return observer
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (obsvbl *channelObservable[V]) UnsubscribeAll() {
	obsvbl.unsubscribeAll()
}

// goPublish to the publishCh and notify observers when values are received.
// This function is blocking and should be run in a goroutine.
func (obsvbl *channelObservable[V]) goPublish() {
	for notification := range obsvbl.publishCh {
		// Copy currentObservers to avoid holding the lock while notifying them.
		// New or existing Observers may (un)subscribe while this notification
		// is being fanned out.
		// The observers at the time of locking, prior to copying, are the canonical
		// set of observers which receive this notification.
		currentObservers := obsvbl.copyObservers()
		for _, obsvr := range currentObservers {
			// TODO_CONSIDERATION: perhaps continue trying to avoid making this
			// notification async as it would effectively use goroutines
			// in memory as a buffer (unbounded).
			obsvr.notify(notification)
		}
	}

	// Here we know that the publisher channel has been closed.
	// Unsubscribe all observers as they can no longer receive notifications.
	obsvbl.unsubscribeAll()
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is done.
// It is a blocking function and intended to be called in a goroutine.
func goUnsubscribeOnDone[V any](ctx context.Context, observer observable.Observer[V]) {
	<-ctx.Done()
	if observer.IsClosed() {
		return
	}
	observer.Unsubscribe()
}
