package notifiable

import (
	"context"
	"fmt"
	"sync"

	"pocket/pkg/observable"
)

var _ observable.Observable[any] = &notifiableObservable[any]{}

type option[V any] func(obs *notifiableObservable[V])

// notifiableObservable implements the observable.Observable interface and can be notified
// via its corresponding notifier channel.
type notifiableObservable[V any] struct {
	notifier    chan V // private channel that is used to emit values to observers
	observersMu sync.RWMutex
	// TODO_THIS_COMMIT: update comment(s)
	// TODO_THIS_COMMIT: consider using interface type
	observers *[]*observer[V] // observers is a list of channels that are subscribed to the observable
}

// NewObservable creates a new observable is notified when the notifier channel
// receives a value.
// func NewObservable[V any](notifier chan V) (observable.Observable[V], chan<- V) {
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	observable := &notifiableObservable[V]{
		observersMu: sync.RWMutex{},
		observers:   new([]*observer[V]),
	}

	for _, opt := range opts {
		opt(observable)
	}

	// If the caller does not provide a notifier, create a new one and return it
	if observable.notifier == nil {
		observable.notifier = make(chan V)
	}

	// Start listening to the notifier and emit values to observers
	go observable.goListen(observable.notifier)

	return observable, observable.notifier
}

func WithNotifier[V any](notifier chan V) option[V] {
	return func(obs *notifiableObservable[V]) {
		obs.notifier = notifier
	}
}

// Subscribe returns an observer which is notified when notifier receives.
func (obs *notifiableObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
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

func (obs *notifiableObservable[V]) Close() {
	obs.close()
}

// TODO_THIS_COMMIT: decide whether this closes the notifier channel; perhaps not
// at oll or only if it was provided...
func (obs *notifiableObservable[V]) close() {
	obs.observersMu.RLock()
	observers := *obs.observers
	obs.observersMu.RUnlock()

	for _, sub := range observers {
		fmt.Printf("notifiableObservable#goListen: unsubscribing %p\n", sub)
		sub.Unsubscribe()
	}

	obs.observersMu.Lock()
	defer obs.observersMu.Unlock()
	obs.observers = new([]*observer[V])
}

// goListen to the notifier and notify observers when values are received. This
// function is blocking and should be run in a goroutine.
func (obs *notifiableObservable[V]) goListen(notifier <-chan V) {
	for notification := range notifier {
		obs.observersMu.RLock()
		observers := *obs.observers
		obs.observersMu.RUnlock()

		for _, sub := range observers {
			// CONSIDERATION: perhaps try to avoid making this notification async
			// as it effectively uses goroutines in memory as a buffer (with
			// little control surface).
			sub.notify(notification)
		}
	}

	// Here we know that the notifier has been closed, all observers should be closed as well
	obs.close()
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is.
// It is blocking and intended to be called in a goroutine.
func goUnsubscribeOnDone[V any](ctx context.Context, subscription observable.Observer[V]) {
	<-ctx.Done()
	subscription.Unsubscribe()
}

// onUnsubscribeFactory returns a function that removes a given observer from the
// observable's list of observers.
func (obs *notifiableObservable[V]) onUnsubscribeFactory() UnsubscribeFunc[V] {
	return func(toRemove *observer[V]) {
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
