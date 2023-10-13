package channel

import (
	"context"
	"fmt"
	"sync"

	"pocket/pkg/observable"
)

var _ observable.Observable[any] = &channelObservable[any]{}

// option is a function which receives and can modify the channelObservable state.
type option[V any] func(obs *channelObservable[V])

// channelObservable implements the observable.Observable interface and can be notified
// via its corresponding producer channel.
type channelObservable[V any] struct {
	// producer is an observable-wide channel that is used to receive values
	// which are subsequently re-sent to observers.
	producer chan V
	// observersMu protects observers from concurrent access/updates
	observersMu sync.RWMutex
	// observers is a list of channelObservers that will be notified with producer
	// receives a value.
	observers *[]*channelObserver[V]
}

// NewObservable creates a new observable is notified when the producer channel
// receives a value.
// func NewObservable[V any](producer chan V) (observable.Observable[V], chan<- V) {
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observer that publishes messages from 1 producer to N observers
	obs := &channelObservable[V]{
		observersMu: sync.RWMutex{},
		observers:   new([]*channelObserver[V]),
	}

	for _, opt := range opts {
		opt(obs)
	}

	// if the caller does not provide a producer, create a new one and return it
	if obs.producer == nil {
		obs.producer = make(chan V)
	}

	// start listening to the producer and emit values to observers
	go obs.goListen(obs.producer)

	return obs, obs.producer
}

// WithProducer returns an option function sets the given producer in an observable
// when passed to NewObservable().
func WithProducer[V any](producer chan V) option[V] {
	return func(obs *channelObservable[V]) {
		obs.producer = producer
	}
}

// Subscribe returns an observer which is notified when the producer channel
// receives a value.
func (obs *channelObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	obs.observersMu.Lock()
	defer func() {
		obs.observersMu.Unlock()
	}()

	observer := NewObserver[V](ctx, obs.onUnsubscribeFactory)

	// caller can rely on context cancellation or call Close() to unsubscribe
	// active observers
	if ctx != nil {
		// asynchronously wait for the context to close and unsubscribe
		go goUnsubscribeOnDone[V](ctx, observer)
	}
	return observer
}

func (obs *channelObservable[V]) Close() {
	obs.close()
}

// CONSIDERATION: decide whether this should close the producer channel; perhaps
// only if it was provided.
func (obs *channelObservable[V]) close() {
	obs.observersMu.RLock()
	observers := *obs.observers
	obs.observersMu.RUnlock()

	for _, sub := range observers {
		fmt.Printf("channelObservable#goListen: unsubscribing %p\n", sub)
		sub.Unsubscribe()
	}

	obs.observersMu.Lock()
	defer obs.observersMu.Unlock()
	obs.observers = new([]*channelObserver[V])
}

// goListen to the producer and notify observers when values are received. This
// function is blocking and should be run in a goroutine.
func (obs *channelObservable[V]) goListen(producer <-chan V) {
	for notification := range producer {
		obs.observersMu.RLock()
		observers := *obs.observers
		obs.observersMu.RUnlock()

		for _, sub := range observers {
			// CONSIDERATION: perhaps continue trying to avoid making this
			// notification async as it would effectively use goroutines
			// in memory as a buffer (with little control surface).
			sub.notify(notification)
		}
	}

	// Here we know that the producer has been closed, all observers should be closed as well
	obs.close()
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is.
// It is blocking and intended to be called in a goroutine.
func goUnsubscribeOnDone[V any](ctx context.Context, subscription observable.Observer[V]) {
	<-ctx.Done()
	subscription.Unsubscribe()
}

// onUnsubscribeFactory returns a function that removes a given channelObserver from the
// observable's list of observers.
func (obs *channelObservable[V]) onUnsubscribeFactory() UnsubscribeFunc[V] {
	return func(toRemove *channelObserver[V]) {
		obs.observersMu.Lock()
		defer obs.observersMu.Unlock()

		for i, subscription := range *obs.observers {
			if subscription == toRemove {
				*obs.observers = append((*obs.observers)[:i], (*obs.observers)[i+1:]...)
				break
			}
			obs.observersMu.Unlock()
		}
	}
}
