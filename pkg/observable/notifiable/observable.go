package notifiable

import (
	"context"
	"sync"

	"pocket/pkg/observable"
)

// Observable implements the observable.Observable interface and can be notified
// via its corresponding notifier channel.
type Observable[V any] struct {
	mu          sync.RWMutex
	ch          <-chan V // private channel that is used to emit values to subscribers
	subscribers []chan V // subscribers is a list of channels that are subscribed to the observable
	closed      bool
}

// NewObservable creates a new observable is notified when the notifier channel
// receives a value.
func NewObservable[V any](notifier chan V) (observable.Observable[V], chan V) {
	// If the caller does not provide a notifier, create a new one and return it
	if notifier == nil {
		notifier = make(chan V)
	}
	notifee := &Observable[V]{sync.RWMutex{}, notifier, []chan V{}, false}

	// Start listening to the notifier and emit values to subscribers
	go notifee.listen(notifier)

	return notifee, notifier
}

// Subscribe gets a subscription to the observable.
func (obs *Observable[V]) Subscribe(ctx context.Context) observable.Subscription[V] {
	obs.mu.Lock()
	defer obs.mu.Unlock()

	// Create a channel for the subscriber and append it to the subscribers list
	ch := make(chan V, 1)
	obs.subscribers = append(obs.subscribers, ch)

	// Removal function used when unsubscribing from the observable
	removeFromObservable := func() {
		obs.mu.Lock()
		defer obs.mu.Unlock()

		for i, s := range obs.subscribers {
			if ch == s {
				obs.subscribers = append(obs.subscribers[:i], obs.subscribers[i+1:]...)
				break
			}
		}
	}

	// Subscription gets its closed state from the observable
	subscription := &Subscription[V]{ch, obs.closed, removeFromObservable}

	go unsubscribeOnDone[V](ctx, subscription)

	return subscription
}

// listen to the notifier and notify subscribers when values are received. This
// function is blocking and should be run in a goroutine.
func (obs *Observable[V]) listen(notifier <-chan V) {
	for v := range notifier {
		// Lock for obs.subscribers slice as it can be modified by subscribers
		obs.mu.RLock()
		for _, ch := range obs.subscribers {
			ch <- v
		}
		obs.mu.RUnlock()
	}

	// Here we know that the notifier has been closed, all subscribers should be closed as well
	obs.mu.Lock()
	obs.closed = true
	for _, ch := range obs.subscribers {
		close(ch)
		obs.subscribers = []chan V{}
	}
	obs.mu.Lock()
}

// unsubscribeOnDone unsubscribes from the subscription when the context is.
// It is blocking and intended to be called in a goroutine.
func unsubscribeOnDone[V any](ctx context.Context, subscription observable.Subscription[V]) {
	if ctx != nil {
		<-ctx.Done()
		subscription.Unsubscribe()
	}
}
