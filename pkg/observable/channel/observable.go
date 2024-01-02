package channel

import (
	"context"
	"sync"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// TODO_DISCUSS: what should this be? should it be configurable? It seems to be most
// relevant in the context of the behavior of the observable when it has multiple
// observers which consume at different rates.
// defaultSubscribeBufferSize is the buffer size of a observable's publish channel.
const defaultPublishBufferSize = 50

var (
	_ observable.Observable[any] = (*channelObservable[any])(nil)
	_ observerManager[any]       = (*channelObservable[any])(nil)
)

// option is a function which receives and can modify the channelObservable state.
type option[V any] func(obs *channelObservable[V])

// channelObservable implements the observable.Observable interface and can be notified
// by sending on its corresponding publishCh channel.
type channelObservable[V any] struct {
	// embed observerManager to encapsulate concurrent-safe read/write access to
	// observers. This also allows higher-level objects to wrap this observable
	// without knowing its specific type by asserting that it implements the
	// observerManager interface.
	observerManager[V]
	// publishCh is an observable-wide channel that is used to receive values
	// which are subsequently fanned out to observers.
	publishCh chan V
}

// NewObservable creates a new observable which is notified when the publishCh
// channel receives a value.
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observable that publishes messages from 1 publishCh to N observers
	obs := &channelObservable[V]{
		observerManager: newObserverManager[V](),
	}

	for _, opt := range opts {
		opt(obs)
	}

	// If the caller does not provide a publishCh, create a new one using the
	// defaultPublishBuffer size and return it.
	if obs.publishCh == nil {
		obs.publishCh = make(chan V, defaultPublishBufferSize)
	}

	// start listening to the publishCh and emit values to observers
	go obs.goPublish()

	return obs, obs.publishCh
}

// WithPublisher returns an option function which sets the given publishCh of the
// resulting observable when passed to NewObservable().
func WithPublisher[V any](publishCh chan V) option[V] {
	return func(obs *channelObservable[V]) {
		obs.publishCh = publishCh
	}
}

// Subscribe returns an observer which is notified when the publishCh channel
// receives a value.
func (obs *channelObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	if ctx == nil {
		ctx = context.Background()
	}

	// caller can cancel context or close the publish channel to unsubscribe active observers
	ctx, cancel := context.WithCancel(ctx)
	removeAndCancel := func(toRemove observable.Observer[V]) {
		obs.observerManager.remove(toRemove)
		cancel()
	}

	// Create a new observer and add it to the list of observers to be notified
	// when publishCh receives a new value.
	observer := NewObserver[V](ctx, removeAndCancel)
	obs.observerManager.add(observer)

	// asynchronously wait for the context to be done and then unsubscribe
	// this observer.
	go obs.observerManager.goUnsubscribeOnDone(ctx, observer)

	return observer
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (obs *channelObservable[V]) UnsubscribeAll() {
	obs.observerManager.removeAll()
}

// goPublish to the publishCh and notify observers when values are received.
// This function is blocking and should be run in a goroutine.
func (obs *channelObservable[V]) goPublish() {
	for notification := range obs.publishCh {
		obs.observerManager.notifyAll(notification)
	}

	// Here we know that the publisher channel has been closed.
	// Unsubscribe all observers as they can no longer receive notifications.
	obs.observerManager.removeAll()
}

// nergeOpt is a function that can be used to modify the behaviour of the
// mergedObservable, during construction.
type mergeOpt[V any] func(mObs *mergedObservable[V])

// observablesToMerge is a struct that holds the observables that are being
// managed by the mergedObservable, along with their index so that they can
// be remoed if they close.
type observablesToMerge[V any] struct {
	index uint64
	obs   observable.Observable[V]
}

// mergedObservable implements the observable.Observable interface and can be
// constructeed in such a way that the supplied observables are merged into
// one observable.
type mergedObservable[V any] struct {
	// observables is a list of all observables that are managed by the
	// mergedObservable.
	observables []observablesToMerge[V]
	// obsMu protects the observables list from concurrent access.
	obsMu sync.RWMutex
	// publishChs is a list of all publish channels for each of the observables
	// managed by the observerManager.
	publishChs []chan<- V
	// mergesObs is the merged observable that emits notifications from each
	// of the observables managed by the observerManager.
	mergedObs observable.Observable[V]
	// megedObsPublishCh is the publish channel for the merged observable.
	mergedObsPublishCh chan<- V
	// delayError indicaates whether the merged observable should delay any
	// error emissions until all observables have completed.
	delayError bool
	// failFast indicates whether the merged observable should fail as soon as
	// any of the observables fail.
	failFast bool
}

// NewMergedObservable constructs a new merged observable instance based on the
// supplied observables and options, by default it will fail fast unless other
// wise specified.
func NewMergedObservable[V any](
	obvserbables []observable.Observable[V],
	opts ...mergeOpt[V],
) (observable.MergedObservable[V], error) {
	// Create the base struct of the merged observable.
	mObs := &mergedObservable[V]{}
	for i, obs := range obvserbables {
		mObs.observables = append(
			mObs.observables, observablesToMerge[V]{
				index: uint64(i),
				obs:   obs,
			})
	}

	// Apply any supplied options to the merged observable.
	for _, opt := range opts {
		opt(mObs)
	}

	// Check that the merged observable has been configured correctly.
	if mObs.failFast && mObs.delayError {
		// We cannot fail fast and delay errors at the same time.
		return nil, observable.ErrMergeObservableMultipleFailModes
	} else if !mObs.failFast && !mObs.delayError {
		// If no fail mode has been specified, default to fail fast.
		mObs.failFast = true
	}

	// Create the merged observable and its publish channel.
	mObs.mergedObs, mObs.mergedObsPublishCh = NewObservable[V]()

	return mObs, nil
}

// WithMergeDelayError is an option provided to the Mapped Observer constructor
// that will delay any error emissons until all observables have completed.
// This means is one or many observables fail the mergerd observable will not
// fail until the last remaining observable(s) have completed.
func WithMergeDelayError[V any](delay bool) mergeOpt[V] {
	return func(mObs *mergedObservable[V]) {
		mObs.delayError = delay
	}
}

// WithFailFast is an option provided to the Mapped Observer constructor that
// will fail the merged observable as soon as any of the observables fail.
// This is the default behaviour of the merged observable.
func WithFailFast[V any](fast bool) mergeOpt[V] {
	return func(mObs *mergedObservable[V]) {
		mObs.failFast = fast
	}
}

// Subscribe returns an observer which merges notifications from all observables
// provided to the merged observable during construction.
func (mObs *mergedObservable[V]) Subscribe(ctx context.Context) observable.Observer[V] {
	mObs.obsMu.RLock()
	defer mObs.obsMu.RUnlock()
	for i, obs := range mObs.observables {
		j := uint64(i)
		go mObs.goMergeObservables(ctx, j, obs.obs)
	}
	return mObs.mergedObs.Subscribe(ctx)
}

// goMergeObservables subscribes to the supplied observable and emits all their
// notifications to the merged observable's publish channel.
func (mObs *mergedObservable[V]) goMergeObservables(
	ctx context.Context,
	idx uint64,
	obs observable.Observable[V],
) {
	// Subscribe to the observable and add the publish channel to the list of
	// publish channels.
	observer := obs.Subscribe(ctx)
	for notification := range observer.Ch() {
		mObs.mergedObsPublishCh <- notification
	}
	// Unsubscribe the observable from the merged observable as it is closed.
	obs.UnsubscribeAll()
	// Remove the observable from the list of observables.
	mObs.observables = append(mObs.observables[:idx], mObs.observables[idx+1:]...)
}

func (mObs *mergedObservable[V]) UnsubscribeAll() {
	// Unsubscribe all observables merged by the merged observable.
	for _, obs := range mObs.observables {
		obs.obs.UnsubscribeAll()
	}
	// Unsubscribe the lustener from the merged observable itself.
	mObs.mergedObs.UnsubscribeAll()
}
