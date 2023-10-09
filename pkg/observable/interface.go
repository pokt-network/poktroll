package observable

import "context"

// Observable is a generic interface that allows multiple subscribers to be
// notified of new values asynchronously.
type Observable[V any] interface {
	Subscribe(context.Context) Subscription[V]
}

// Subscription is a generic interface that provides access to the notified
// channel and allows unsubscribing from an observable.
type Subscription[V any] interface {
	Unsubscribe()
	Ch() <-chan V
}
