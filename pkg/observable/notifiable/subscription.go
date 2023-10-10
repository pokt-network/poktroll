package notifiable

import "pocket/pkg/observable"

var _ observable.Subscription[any] = &Subscription[any]{}

// Subscription implements the observable.Subscription interface.
type Subscription[V any] struct {
	ch                   chan V
	closed               bool
	removeFromObservable func()
}

// Unsubscribe closes the subscription channel and removes the subscription from
// the observable.
func (sub *Subscription[V]) Unsubscribe() {
	if sub.closed {
		return
	}

	close(sub.ch)
	sub.closed = true
	sub.removeFromObservable()
}

// Ch returns the subscription channel.
func (sub *Subscription[V]) Ch() <-chan V {
	return sub.ch
}
