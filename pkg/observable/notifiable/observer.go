package notifiable

import (
	"context"
	"fmt"
	"sync"

	"pocket/pkg/observable"
)

// TODO_THIS_COMMIT: explain why buffer size is 1
// observerBufferSize ...
const observerBufferSize = 1

var _ observable.Observer[any] = &observer[any]{}

// observer implements the observable.Observer interface.
type observer[V any] struct {
	ctx        context.Context
	observerMu *sync.RWMutex
	observerCh chan V
	// TODO_THIS_COMMIT: add comment
	onUnsubscribe func(toRemove *observer[V])
	closed        bool
}

type UnsubscribeFactory[V any] func() UnsubscribeFunc[V]
type UnsubscribeFunc[V any] func(toRemove *observer[V])

func NewSubscription[V any](
	ctx context.Context,
	onUnsubscribeFactory UnsubscribeFactory[V],
) *observer[V] {
	// Create a channel for the subscriber and append it to the observers list
	ch := make(chan V, 1)
	fmt.Printf("notifiableObservable#Subscribe: opening %p\n", ch)

	return &observer[V]{
		ctx:           ctx,
		observerMu:    new(sync.RWMutex),
		observerCh:    make(chan V, observerBufferSize),
		onUnsubscribe: onUnsubscribeFactory(),
	}
}

// Unsubscribe closes the subscription channel and removes the subscription from
// the observable.
func (obv *observer[V]) Unsubscribe() {
	obv.observerMu.Lock()
	defer func() {
		obv.observerMu.Unlock()
	}()

	if obv.closed {
		return
	}

	fmt.Printf("observer#Unsubscribe: closing %p\n", obv.observerCh)
	close(obv.observerCh)
	obv.closed = true

	obv.onUnsubscribe(obv)
}

// Ch returns a receive-only subscription channel.
func (obv *observer[V]) Ch() <-chan V {
	obv.observerMu.Lock()
	defer func() {
		obv.observerMu.Unlock()
	}()

	return obv.observerCh
}

// TODO_CLEANUP_COMMENT: used by observable to send to subscriber  channel
// because observer#Ch returns a receive-only channel
func (obv *observer[V]) notify(value V) {
	obv.observerMu.Lock()
	ch, closed := obv.observerCh, obv.closed
	//defer func() {
	//	obv.observersMu.Unlock()
	//}()

	if closed {
		obv.observerMu.Unlock()
		return
	}
	obv.observerMu.Unlock()

	select {
	case ch <- value:
	case <-obv.ctx.Done():
	}
}
