package channel

import (
	"sync"

	"pocket/pkg/observable"
)

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

type channelObservableInternals[V any] struct {
	// observersMu protects observers from concurrent access/updates
	observersMu *sync.RWMutex
	// observers is a list of channelObservers that will be notified when publishCh
	// receives a new value.
	observers []*channelObserver[V]
}

func newObservableInternals[V any]() channelObservableInternals[V] {
	return channelObservableInternals[V]{
		observersMu: &sync.RWMutex{},
		observers:   []*channelObserver[V]{},
	}
}

// addObserver implements the respective member of observableInternals. It is used
// by the channelObservable implementation as well as embedders of observableInternals
// (e.g. replayObservable).
// It panics if toAdd is not a channelObserver.
func (coi *channelObservableInternals[V]) addObserver(
	toAdd observable.Observer[V],
) {
	// must (write) lock observersMu so that we can safely append to the observers list
	coi.observersMu.Lock()
	defer coi.observersMu.Unlock()

	coi.observers = append(coi.observers, toAdd.(*channelObserver[V]))
}

// onUnsubscribe returns a function that removes a given observer from the
// observable's list of observers.
// It implements the respective member of observableInternals and is used by
// the channelObservable implementation as well as embedders of observableInternals
// (e.g. replayObservable).
func (coi *channelObservableInternals[V]) onUnsubscribe(
	toRemove observable.Observer[V],
) {
	// must (write) lock to iterate over and modify the observers list
	coi.observersMu.Lock()
	defer coi.observersMu.Unlock()

	for i, observer := range coi.observers {
		if observer == toRemove {
			coi.observers = append((coi.observers)[:i], (coi.observers)[i+1:]...)
			break
		}
	}
}

// unsubscribeAll unsubscribes and removes all observers from the observable.
// It implements the respective member of observableInternals and is used by
// the channelObservable implementation as well as embedders of observableInternals
// (e.g. replayObservable).
func (coi *channelObservableInternals[V]) unsubscribeAll() {
	// Copy currentObservers to avoid holding the lock while unsubscribing them.
	// The observers at the time of locking, prior to copying, are the canonical
	// set of observers which are unsubscribed.
	// New or existing Observers may (un)subscribe while the observable is closing.
	// Any such observers won't be isClosed but will also stop receiving notifications
	// immediately (if they receive any at all).
	currentObservers := coi.copyObservers()
	for _, observer := range currentObservers {
		observer.Unsubscribe()
	}

	// Reset observers to an empty list. This purges any observers which might have
	// subscribed while the observable was closing.
	coi.observersMu.Lock()
	coi.observers = []*channelObserver[V]{}
	coi.observersMu.Unlock()
}

// copyObservers returns a copy of the current observers list. It is safe to
// call concurrently.
func (coi *channelObservableInternals[V]) copyObservers() (observers []*channelObserver[V]) {
	defer coi.observersMu.RUnlock()

	// This loop blocks on acquiring a read lock on observersMu. If TryRLock
	// fails, the loop continues until it succeeds. This is intended to give
	// callers a guarantee that this copy operation won't contribute to a deadlock.
	for {
		// block until a read lock can be acquired
		if coi.observersMu.TryRLock() {
			break
		}
	}

	observers = make([]*channelObserver[V], len(coi.observers))
	copy(observers, coi.observers)

	return observers
}
