package channel

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

var _ observable.MergedObservable[any] = (*mergedObservable[any])(nil)

// nergeOpt is a function that can be used to modify the behaviour of the
// mergedObservable, during construction.
type mergeOpt[V any] func(mObs *mergedObservable[V])

type mergedObservable[V any] struct {
	// observables is a list of all observables that are managed by the
	// mergedObservable.
	observables []observable.Observable[V]
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
	mObs := &mergedObservable[V]{
		observables: obvserbables,
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
	mObs.mergedObs, mObs.mergedObsPublishCh = channel.NewObservable[V]()

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
	for _, obs := range mObs.observables {
		go mObs.goMergeObservables(ctx, obs)
	}
	return mObs.mergedObs.Subscribe(ctx)
}

// goMergeObservables subscribes to the supplied observable and emits all their
// notifications to the merged observable's publish channel.
func (mObs *mergedObservable) goMergeObservables[V](ctx, obs observable.Observable[V]) {
	// Subscribe to the observable and add the publish channel to the list of
	// publish channels.
	observer := obs.Subscribe(ctx)
	for notification := range observer.Ch() {
		mObs.mergedObsPublishCh <- notification
	}
}

func (mObs *mergedObservable[V]) UnsubscribeAll() {
	// Unsubscribe all observables merged by the merged observable.
	for _, obs := range mObs.observables {
		obs.UnsubscribeAll()
	}
	// Unsubscribe the lustener from the merged observable itself.
	mObs.mergedObs.UnsubscribeAll()
}
