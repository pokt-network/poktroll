package observable

import "context"

// NOTE: We explicitly decided to write a small and custom notifications package
// to keep logic simple and minimal. If the needs & requirements of this library ever
// grow, other packages (e.g. https://github.com/ReactiveX/RxGo) can be considered.
// (see: https://github.com/ReactiveX/RxGo/pull/377)

// ReplayObservable is an observable which replays the last n values published
// to new observers, before publishing new values to observers.
type ReplayObservable[V any] interface {
	Observable[V]
	// Last synchronously returns the last n values from the replay buffer.
	Last(ctx context.Context, n int) []V
}

// Observable is a generic interface that allows multiple subscribers to be
// notified of new values asynchronously.
// It is analogous to a publisher in a "Fan-Out" system design.
type Observable[V any] interface {
	// Next synchronously returns the next value from the observable.
	Next(context.Context) V
	// Subscribe returns an observer which is notified when the publishCh channel
	// receives a value.
	Subscribe(context.Context) Observer[V]
	// UnsubscribeAll unsubscribes and removes all observers from the observable.
	UnsubscribeAll()
}

// Observer is a generic interface that provides access to the notified
// channel and allows unsubscribing from an Observable.
// It is analogous to a subscriber in a "Fan-Out" system design.
type Observer[V any] interface {
	// Unsubscribe closes the subscription channel and removes the subscription from
	// the observable.
	Unsubscribe()
	// Ch returns a receive-only subscription channel.
	Ch() <-chan V
	// IsClosed returns true if the observer has been unsubscribed.
	// A closed observer cannot be reused.
	IsClosed() bool
}
