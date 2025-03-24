package channel

import (
	"context"
	"sync"

	"github.com/pokt-network/pocket/pkg/observable"
)

var _ observerManager[any] = (*channelObserverManager[any])(nil)

// observerManager is an interface intended to be used between an observable and some
// higher-level abstraction and/or observable implementation which would embed it.
// Embedding this interface rather than a channelObservable directly allows for
// more transparency and flexibility in higher-level code.
// NOTE: this interface MUST be used with a common concrete Observer type.
// TODO_CONSIDERATION: Consider whether `observerManager` and `Observable` should remain as separate
// types after some more time and experience using both.
type observerManager[V any] interface {
	notifyAll(notification V)
	add(toAdd observable.Observer[V])
	remove(toRemove observable.Observer[V])
	removeAll()
	goUnsubscribeOnDone(ctx context.Context, observer observable.Observer[V])
}

// TODO_CONSIDERATION: if this were a generic implementation, we wouldn't need
// to cast `toAdd` to a channelObserver in add. There are two things
// currently preventing a generic observerManager implementation:
// 1. channelObserver#notify() is not part of the observable.Observer interface
// 	  and is therefore not accessible here. If we move everything into the
//	  `observable` pkg so that the unexported member is in scope, then the channel
//	  pkg can't implement it for the same reason, it's an unexported method defined
//	  in a different pkg.
// 2. == is not defined for a generic Observer type. We would have to add an Equals()
// 	  to  the Observer interface.

// channelObserverManager implements the observerManager interface using
// channelObservers.
type channelObserverManager[V any] struct {
	// observersMu protects observers from concurrent access/updates
	observersMu *sync.RWMutex
	// observers is a list of channelObservers that will be notified when new value
	// are received.
	observers []*channelObserver[V]
}

func newObserverManager[V any]() *channelObserverManager[V] {
	return &channelObserverManager[V]{
		observersMu: &sync.RWMutex{},
		observers:   make([]*channelObserver[V], 0),
	}
}

func (com *channelObserverManager[V]) notifyAll(notification V) { //nolint:unused // Used in the observable implementation.
	// Copy currentObservers to avoid holding the lock while notifying them.
	// New or existing Observers may (un)subscribe while this notification
	// is being fanned out.
	// The observers at the time of locking, prior to copying, are the canonical
	// set of observers which receive this notification.
	currentObservers := com.copyObservers()
	for _, obsvr := range currentObservers {
		// TODO_TECHDEBT: since this synchronously notifies all observers in a loop,
		// it is possible to block here, part-way through notifying all observers,
		// on a slow observer consumer (i.e. full buffer). Instead, we should notify
		// observers with some limited concurrency of "worker" goroutines.
		// The storj/common repo contains such a `Limiter` implementation, see:
		// https://github.com/storj/common/blob/main/sync2/limiter.go.
		obsvr.notify(notification)
	}
}

// addObserver implements the respective member of observerManager. It is used
// by the channelObservable implementation as well as embedders of observerManager
// (e.g. replayObservable).
// It panics if toAdd is not a channelObserver.
func (com *channelObserverManager[V]) add(toAdd observable.Observer[V]) { //nolint:unused // Used in the observable implementation.
	// must (write) lock observersMu so that we can safely append to the observers list
	com.observersMu.Lock()
	defer com.observersMu.Unlock()

	com.observers = append(com.observers, toAdd.(*channelObserver[V]))
}

// remove removes a given observer from the observable's list of observers.
// It implements the respective member of observerManager and is used by
// the channelObservable implementation as well as embedders of observerManager
// (e.g. replayObservable).
func (com *channelObserverManager[V]) remove(toRemove observable.Observer[V]) { //nolint:unused // Used in the observable implementation.
	// must (write) lock to iterate over and modify the observers list
	com.observersMu.Lock()
	defer com.observersMu.Unlock()

	for i, observer := range com.observers {
		if observer == toRemove {
			com.observers = append((com.observers)[:i], (com.observers)[i+1:]...)
			break
		}
	}
}

// removeAll unsubscribes and removes all observers from the observable.
// It implements the respective member of observerManager and is used by
// the channelObservable implementation as well as embedders of observerManager
// (e.g. replayObservable).
func (com *channelObserverManager[V]) removeAll() { //nolint:unused // Used in the observable implementation.
	// Copy currentObservers to avoid holding the lock while unsubscribing them.
	// The observers at the time of locking, prior to copying, are the canonical
	// set of observers which are unsubscribed.
	// New or existing Observers may (un)subscribe while the observable is closing.
	// Any such observers won't be isClosed but will also stop receiving notifications
	// immediately (if they receive any at all).
	currentObservers := com.copyObservers()
	for _, observer := range currentObservers {
		observer.Unsubscribe()
	}

	// Reset observers to an empty list. This purges any observers which might have
	// subscribed while the observable was closing.
	com.observersMu.Lock()
	com.observers = []*channelObserver[V]{}
	com.observersMu.Unlock()
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is done.
// It is a blocking function and intended to be called in a goroutine.
func (com *channelObserverManager[V]) goUnsubscribeOnDone( //nolint:unused // Used in the observable implementation.
	ctx context.Context,
	observer observable.Observer[V],
) { //nolint:unused // Used in the observable implementation.
	<-ctx.Done()
	if observer.IsClosed() {
		return
	}
	observer.Unsubscribe()
}

// copyObservers returns a copy of the current observers list. It is safe to
// call concurrently. Notably, it is not part of the observerManager interface.
func (com *channelObserverManager[V]) copyObservers() (observers []*channelObserver[V]) { //nolint:unused // Used in the observable implementation.
	defer com.observersMu.RUnlock()

	// This loop blocks on acquiring a read lock on observersMu. If TryRLock
	// fails, the loop continues until it succeeds. This is intended to give
	// callers a guarantee that this copy operation won't contribute to a deadlock.
	com.observersMu.RLock()

	observers = make([]*channelObserver[V], len(com.observers))
	copy(observers, com.observers)

	return observers
}
