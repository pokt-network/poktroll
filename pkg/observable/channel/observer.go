package channel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pocket/pkg/observable"
)

const (
	// DISCUSS: what should this be? should it be configurable? It seems to be most
	// relevant in the context of the behavior of the observable when it has multiple
	// observers which consume at different rates.
	// observerBufferSize is the buffer size of a channelObserver's channel.
	observerBufferSize = 1
	// sendRetryInterval is the duration between attempts to send on the observer's
	// channel. It facilitates a branch in a for loop which unlocks the observer's
	// mutex and tries again.
	// NOTE: setting this too low can cause the send retry loop to "slip", giving
	// up on a send attempt before the channel is ready to receive for multiple
	// iterations of the loop.
	sendRetryInterval = 100 * time.Millisecond
)

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
	obsvr.unsubscribe()
}

// Ch returns a receive-only subscription channel.
func (obsvr *channelObserver[V]) Ch() <-chan V {
	//fmt.Println("obssvr#Ch:80 locking")
	////for {
	////	if obsvr.observerMu.TryRLock() {
	////		break
	////	}
	////	fmt.Println("retrying")
	////	time.Sleep(30 * time.Millisecond)
	////}
	////obsvr.observerMu.RLock()
	//defer func() {
	//	fmt.Println("obssvr#Ch:83 unlocking")
	//	//obsvr.observerMu.RUnlock()
	//}()

	return obsvr.observerCh
}

func (obsvr *channelObserver[V]) unsubscribe() {
	fmt.Println("[obsersverMu] unsubscribe locking...")
	obsvr.observerMu.Lock()
	fmt.Println("[obsersverMu] ...unsubscribe locked")
	//defer func() {
	//	obsvr.observerMu.Unlock()
	//}()

	if obsvr.closed {
		//fmt.Println("[obsersverMu] unsubscribe unlocking (closed)...")
		obsvr.observerMu.Unlock()
		//fmt.Println("[obsersverMu] ...unsubscribe unlocked (closed)")
		return
	}

	fmt.Printf("[obsersverMu] channelObserver#Unsubscribe: closing %p\n", obsvr.observerCh)
	close(obsvr.observerCh)
	obsvr.closed = true
	//fmt.Println("[obsersverMu] unsubscribe unlocking (open)...")
	obsvr.observerMu.Unlock()
	//fmt.Println("[obsersverMu] ...unsubscribe unlocked (open)")

	obsvr.onUnsubscribe(obsvr)
}

// notify is used called by observable to send on the observer channel. Can't
// use channelObserver#Ch because it's receive-only.
func (obsvr *channelObserver[V]) notify(value V) {
	//valueStr := fmt.Sprintf("%s", value)
	//fmt.Printf("notify called, value: %s\n", valueStr)

	// TODO_THIS_COMMIT: prove the need for the send retry loop via tests.
	sendRetryTicker := time.NewTicker(sendRetryInterval)
	// wait sendRetryInterval before releasing the lock and trying again.
	for {
		//fmt.Println(valueStr)
		//if valueStr == "message-9" {
		//	fmt.Println("on message-9")
		//}
		obsvr.observerMu.RLock()
		if obsvr.closed {
			//obsvr.observerMu.RUnlock()
			return
		}
		//obsvr.observerMu.RUnlock()

		select {
		case <-obsvr.ctx.Done():
			fmt.Println("ctx done!")
			obsvr.observerMu.RUnlock()
			// TECHDEBT: add a  default path which buffers values so that the sender
			// doesn't block and other consumers can still receive.
			// TECHDEBT: add some logic to drain the buffer at some appropriate time

			// TODO_THIS_COMMIT: is this redundant?
			//obsvr.unsubscribe()
			return
		case obsvr.observerCh <- value:
			obsvr.observerMu.RUnlock()
			//fmt.Printf("sending value: %s\n", valueStr)
			return
		//default:
		case <-sendRetryTicker.C:
			// if channel is blocked,
			fmt.Println("send loop looping")
			obsvr.observerMu.RUnlock()
		}

		time.Sleep(sendRetryInterval)
	}
}
