package channel

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// defaultPublishBufferSize is the buffer size of a observable's publish channel.
//
// DEV_NOTE: This was increased from 50 to 1_000 to prevent "missing supplier operator signature" errors during high load.
// The relay mining pipeline needs breathing room when processing spikes to avoid blocking channel sends
// that cause request timeouts during signature generation.
//
// TODO: Consider making this configurable via RelayMiner config for high-throughput deployments
const defaultPublishBufferSize = 1_000

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
	// subscribeBufferSize is the buffer size of each observer channel created by
	// Subscribe. Defaults to defaultSubscribeBufferSize; override via
	// WithSubscribeBufferSize for high-throughput pipelines that need more
	// breathing room between the publisher and a slow consumer.
	subscribeBufferSize int
}

// NewObservable creates a new observable which is notified when the publishCh
// channel receives a value.
func NewObservable[V any](opts ...option[V]) (observable.Observable[V], chan<- V) {
	// initialize an observable that publishes messages from 1 publishCh to N observers
	obs := &channelObservable[V]{
		observerManager:     newObserverManager[V](),
		subscribeBufferSize: defaultSubscribeBufferSize,
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

// WithPublishBufferSize returns an option function which sets the buffer size of
// the observable's publish channel. Use this instead of WithPublisher when only
// the buffer size (not a pre-existing channel) needs to be customized.
// A larger buffer absorbs bigger producer bursts before sends block/drop.
func WithPublishBufferSize[V any](size int) option[V] {
	return func(obs *channelObservable[V]) {
		obs.publishCh = make(chan V, size)
	}
}

// WithSubscribeBufferSize returns an option function which sets the buffer size of
// each observer channel created by the observable's Subscribe method.
// A larger buffer gives a slow consumer more slack before it stalls the publisher
// (and, transitively, the upstream pipeline).
func WithSubscribeBufferSize[V any](size int) option[V] {
	return func(obs *channelObservable[V]) {
		obs.subscribeBufferSize = size
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
		obs.remove(toRemove)
		cancel()
	}

	// Create a new observer and add it to the list of observers to be notified
	// when publishCh receives a new value.
	observer := NewObserver[V](ctx, removeAndCancel, obs.subscribeBufferSize)
	obs.add(observer)

	// asynchronously wait for the context to be done and then unsubscribe
	// this observer.
	go obs.goUnsubscribeOnDone(ctx, observer)

	return observer
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (obs *channelObservable[V]) UnsubscribeAll() {
	obs.removeAll()
}

// goPublish to the publishCh and notify observers when values are received.
// This function is blocking and should be run in a goroutine.
func (obs *channelObservable[V]) goPublish() {
	for notification := range obs.publishCh {
		obs.notifyAll(notification)
	}

	// Here we know that the publisher channel has been closed.
	// Unsubscribe all observers as they can no longer receive notifications.
	obs.removeAll()
}
