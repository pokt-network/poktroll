package channel

import "pocket/pkg/observable"

// observableInternals is an interface intended to be used between an observable and some
// higher-level abstraction and/or observable implementation which would embed it.
// Embedding this interface rather than a channelObservable directly allows for
// more transparency and flexibility in higher-level code.
// NOTE: this interface MUST be used with a common concrete Observer type.
type observableInternals[V any] interface {
	addObserver(toAdd observable.Observer[V])
	onUnsubscribe(toRemove observable.Observer[V])
	unsubscribeAll()
}
