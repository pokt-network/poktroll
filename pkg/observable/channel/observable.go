package channel

import (
	"context"
	"sync"

	"pocket/pkg/observable"
)

// TODO_DISCUSS: what should this be? should it be configurable? It seems to be most
// relevant in the context of the behavior of the observable when it has multiple
// observers which consume at different rates.
// defaultSubscribeBufferSize is the buffer size of a observable's publish channel.
const defaultPublishBufferSize = 50

var _ observable.Observable[any] = &channelObservable[any]{}

// option is a function which receives and can modify the channelObservable state.
type option[V any] func(obs *channelObservable[V])

// channelObservable implements the observable.Observable interface and can be notified
// by sending on its corresponding publishCh channel.
type channelObservable[V any] struct {
	// publishCh is an observable-wide channel that is used to receive values
	// which are subsequently fanned out to observers.
	publishCh chan V
	// observersMu protects observers from concurrent access/updates
	observersMu *sync.RWMutex
	// observers is a list of channelObservers that will be notified when publishCh
	// receives a new value.
	observers []*channelObserver[V]
}

// NewObservable creates a new observable which is notified when the publishCh
// channel receives a value.
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observable that publishes messages from 1 publishCh to N observers
	obs := &channelObservable[V]{
		observersMu: &sync.RWMutex{},
		observers:   []*channelObserver[V]{},
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

func (obsvbl *channelObservable[V]) Next(ctx context.Context) V {
	tempObserver := obsvbl.Subscribe(ctx)
	defer tempObserver.Unsubscribe()

	val := <-tempObserver.Ch()
	return val
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (obsvbl *channelObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	// must (write) lock observersMu so that we can safely append to the observers list
	obsvbl.observersMu.Lock()
	defer obsvbl.observersMu.Unlock()

	observer := NewObserver[V](ctx, obsvbl.onUnsubscribe)
	obsvbl.observers = append(obsvbl.observers, observer)

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

// unsubscribeAll unsubscribes and removes all observers from the observable.
func (obsvbl *channelObservable[V]) unsubscribeAll() {
	// Copy currentObservers to avoid holding the lock while unsubscribing them.
	// The observers at the time of locking, prior to copying, are the canonical
	// set of observers which are unsubscribed.
	// New or existing Observers may (un)subscribe while the observable is closing.
	// Any such observers won't be isClosed but will also stop receiving notifications
	// immediately (if they receive any at all).
	currentObservers := obsvbl.copyObservers()
	for _, observer := range currentObservers {
		observer.Unsubscribe()
	}

	// Reset observers to an empty list. This purges any observers which might have
	// subscribed while the observable was closing.
	obsvbl.observersMu.Lock()
	obsvbl.observers = []*channelObserver[V]{}
	obsvbl.observersMu.Unlock()
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

// copyObservers returns a copy of the current observers list. It is safe to
// call concurrently.
func (obsvbl *channelObservable[V]) copyObservers() (observers []*channelObserver[V]) {
	defer obsvbl.observersMu.RUnlock()

	// This loop blocks on acquiring a read lock on observersMu. If TryRLock
	// fails, the loop continues until it succeeds. This is intended to give
	// callers a guarantee that this copy operation won't contribute to a deadlock.
	for {
		// block until a read lock can be acquired
		if obsvbl.observersMu.TryRLock() {
			break
		}
	}

	observers = make([]*channelObserver[V], len(obsvbl.observers))
	copy(observers, obsvbl.observers)

	return observers
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

// onUnsubscribe returns a function that removes a given observer from the
// observable's list of observers.
func (obsvbl *channelObservable[V]) onUnsubscribe(toRemove *channelObserver[V]) {
	// must (write) lock to iterate over and modify the observers list
	obsvbl.observersMu.Lock()
	defer obsvbl.observersMu.Unlock()

	for i, observer := range obsvbl.observers {
		if observer == toRemove {
			obsvbl.observers = append((obsvbl.observers)[:i], (obsvbl.observers)[i+1:]...)
			break
		}
	}
}
