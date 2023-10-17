package channel

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	observersMu *sync.RWMutex
	// observers is a list of channelObservers that will be notified when producer
	// receives a value.
	observers []*channelObserver[V]
}

// NewObservable creates a new observable is notified when the producer channel
// receives a value.
// func NewObservable[V any](producer chan V) (observable.Observable[V], chan<- V) {
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observer that publishes messages from 1 producer to N observers
	obs := &channelObservable[V]{
		observersMu: &sync.RWMutex{},
		observers:   []*channelObserver[V]{},
	}

	for _, opt := range opts {
		opt(obs)
	}

	// if the caller does not provide a producer, create a new one and return it
	if obs.producer == nil {
		obs.producer = make(chan V)
	}

	// start listening to the producer and emit values to observers
	go obs.goProduce(obs.producer)

	return obs, obs.producer
}

// WithProducer returns an option function which sets the given producer of the
// resulting observable when passed to NewObservable().
func WithProducer[V any](producer chan V) option[V] {
	return func(obs *channelObservable[V]) {
		obs.producer = producer
	}
}

// Subscribe returns an observer which is notified when the producer channel
// receives a value.
func (obsvbl *channelObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	// must lock observersMu so that we can safely append to the observers list
	obsvbl.observersMu.Lock()
	defer obsvbl.observersMu.Unlock()

	observer := NewObserver[V](ctx, obsvbl.onUnsubscribe)
	obsvbl.observers = append(obsvbl.observers, observer)

	// caller can rely on context cancellation or call Close() to unsubscribe
	// active observers
	if ctx != nil {
		// asynchronously wait for the context to close and unsubscribe
		go goUnsubscribeOnDone[V](ctx, observer)
	}
	return observer
}

func (obsvbl *channelObservable[V]) Close() {
	obsvbl.close()
}

// CONSIDERATION: decide whether this should close the producer channel; perhaps
// only if it was provided.
func (obsvbl *channelObservable[V]) close() {
	// must lock in order to copy the observers list
	obsvbl.observersMu.Lock()
	// copy observers to avoid holding the lock while unsubscribing them
	var activeObservers = make([]*channelObserver[V], len(obsvbl.observers))
	for idx, toClose := range obsvbl.observers {
		activeObservers[idx] = toClose
	}
	// unlock before unsubscribing to avoid deadlock
	obsvbl.observersMu.Unlock()

	for _, observer := range activeObservers {
		observer.Unsubscribe()
	}

	// clear observers
	obsvbl.observersMu.Lock()
	obsvbl.observers = []*channelObserver[V]{}
	obsvbl.observersMu.Unlock()
}

// goProduce to the producer and notify observers when values are received. This
// function is blocking and should be run in a goroutine.
func (obsvbl *channelObservable[V]) goProduce(producer <-chan V) {
	var observers []*channelObserver[V]
	for notification := range producer {
		//fmt.Printf("producer received notification: %s\n", notification)
		// TODO_THIS_COMMIT: (dis)prove the need for this in a test
		// copy observers to avoid holding the lock while notifying
		for {
			//fmt.Println("[obsersversMu] goProduce Rlocking...")
			if !obsvbl.observersMu.TryRLock() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			observers = make([]*channelObserver[V], len(obsvbl.observers))
			//obsvbl.observersMu.RLock()
			//observers := make([]*channelObserver[V], len(obsvbl.observers))
			for i, obsvr := range obsvbl.observers {
				observers[i] = obsvr
			}
			obsvbl.observersMu.RUnlock()
			break
		}

		// notify observers
		for _, obsvr := range observers {
			// CONSIDERATION: perhaps continue trying to avoid making this
			// notification async as it would effectively use goroutines
			// in memory as a buffer (with little control surface).
			obsvr.notify(notification)
		}
	}

	// Here we know that the producer has been closed, all observers should be closed as well
	obsvbl.close()
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is.
// It is blocking and intended to be called in a goroutine.
func goUnsubscribeOnDone[V any](ctx context.Context, subscription observable.Observer[V]) {
	<-ctx.Done()
	fmt.Println("goUnsubscribeOnDone: context done")
	subscription.Unsubscribe()
}

// onUnsubscribe returns a function that removes a given channelObserver from the
// observable's list of observers.
func (obsvbl *channelObservable[V]) onUnsubscribe(toRemove *channelObserver[V]) {
	// must lock to iterato over and modify observers list
	obsvbl.observersMu.Lock()
	defer obsvbl.observersMu.Unlock()

	for i, observer := range obsvbl.observers {
		if observer == toRemove {
			obsvbl.observers = append((obsvbl.observers)[:i], (obsvbl.observers)[i+1:]...)
			break
		}
	}
}
