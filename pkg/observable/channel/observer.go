package channel

import (
	"context"
	"fmt"
	"sync"

	"pocket/pkg/observable"
)

// TODO_THIS_COMMIT: explain why buffer size is 1
// observerBufferSize ...
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
func (obv *channelObserver[V]) Unsubscribe() {
	obv.observerMu.Lock()
	defer func() {
		obv.observerMu.Unlock()
	}()

	if obv.closed {
		return
	}

	fmt.Printf("channelObserver#Unsubscribe: closing %p\n", obv.observerCh)
	close(obv.observerCh)
	obv.closed = true

	obv.onUnsubscribe(obv)
}

// Ch returns a receive-only subscription channel.
func (obv *channelObserver[V]) Ch() <-chan V {
	obv.observerMu.Lock()
	defer func() {
		obv.observerMu.Unlock()
	}()

	return obv.observerCh
}

// TODO_CLEANUP_COMMENT: used by observable to send to subscriber  channel
// because channelObserver#Ch returns a receive-only channel
func (obv *channelObserver[V]) notify(value V) {
	obv.observerMu.Lock()
	ch, closed := obv.observerCh, obv.closed
	defer obv.observerMu.Unlock()

	if closed {
		return
	}

	select {
	case ch <- value:
	case <-obv.ctx.Done():
		// TECHDEBT: add a  default path which buffers values so that the sender
		// doesn't block and other consumers can still receive.
		// TECHDEBT: add some logic to drain the buffer at some appropriate time
	}
}
