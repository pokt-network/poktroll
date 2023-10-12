package observable

import "context"

// Observable is a generic interface that allows multiple subscribers to be
// notified of new values asynchronously.
type Observable[V any] interface {
	Subscribe(context.Context) Observer[V]
	Close()
}

// Observer is a generic interface that provides access to the notified
// channel and allows unsubscribing from an observable.
type Observer[V any] interface {
	Unsubscribe()
	Ch() <-chan V
}
