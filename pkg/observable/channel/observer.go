package channel

import (
	"context"
	"fmt"
	"sync"

	"pocket/pkg/observable"
)

// DISCUSS: what should this be? should it be configurable? It seems to be most
// relevant in the context of the behavior of the observable when it has multiple
// observers which consume at different rates.
// observerBufferSize is the buffer size of a channelObserver's channel.
const observerBufferSize = 1

var _ observable.Observer[any] = &channelObserver[any]{}

// channelObserver implements the observable.Observer interface.
type channelObserver[V any] struct {
	ctx context.Context
	// onUnsubscribe is called in Observer#Unsubscribe, removing the respective
	// observer from observers in a concurrency-safe manner.
	onUnsubscribe func(toRemove *channelObserver[V])
	// observerMu protects the observerCh and closed fields.
	observerMu *sync.RWMutex
	// observerCh is the channel that is used to emit values to the observer.
	// I.e. on the "N" side of the 1:N relationship between observable and
	// observer.
	observerCh chan V
	// closed indicates whether the observer has been closed. It's set in
	// unsubscribe; closed observers can't be reused.
	closed bool
}

type UnsubscribeFactory[V any] func() UnsubscribeFunc[V]
type UnsubscribeFunc[V any] func(toRemove *channelObserver[V])

func NewObserver[V any](
	ctx context.Context,
	onUnsubscribeFactory UnsubscribeFactory[V],
) *channelObserver[V] {
	// Create a channel for the subscriber and append it to the observers list
	ch := make(chan V, 1)
	fmt.Printf("channelObservable#Subscribe: opening %p\n", ch)

	return &channelObserver[V]{
		ctx:           ctx,
		observerMu:    new(sync.RWMutex),
		observerCh:    make(chan V, observerBufferSize),
		onUnsubscribe: onUnsubscribeFactory(),
	}
}

// Unsubscribe closes the subscription channel and removes the subscription from
// the observable.
func (obsvr *channelObserver[V]) Unsubscribe() {
	obsvr.observerMu.Lock()
	defer func() {
		obsvr.observerMu.Unlock()
	}()

	if obsvr.closed {
		return
	}

	fmt.Printf("channelObserver#Unsubscribe: closing %p\n", obsvr.observerCh)
	close(obsvr.observerCh)
	obsvr.closed = true

	obsvr.onUnsubscribe(obsvr)
}

// Ch returns a receive-only subscription channel.
func (obsvr *channelObserver[V]) Ch() <-chan V {
	obsvr.observerMu.Lock()
	defer func() {
		obsvr.observerMu.Unlock()
	}()

	return obsvr.observerCh
}

// TODO_CLEANUP_COMMENT: used by observable to send to subscriber  channel
// because channelObserver#Ch returns a receive-only channel
func (obsvr *channelObserver[V]) notify(value V) {
	obsvr.observerMu.Lock()
	defer obsvr.observerMu.Unlock()

	if obsvr.closed {
		return
	}

	select {
	case obsvr.observerCh <- value:
	case <-obsvr.ctx.Done():
		// TECHDEBT: add a  default path which buffers values so that the sender
		// doesn't block and other consumers can still receive.
		// TECHDEBT: add some logic to drain the buffer at some appropriate time
	}
}
