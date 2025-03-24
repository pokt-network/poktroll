package channel

import (
	"context"
	"sync"
	"time"

	"github.com/pokt-network/pocket/pkg/observable"
	"github.com/pokt-network/pocket/pkg/polylog"
	_ "github.com/pokt-network/pocket/pkg/polylog/polyzero"
)

const (
	// TODO_DISCUSS: what should this be? should it be configurable? It seems to be most
	// relevant in the context of the behavior of the observable when it has multiple
	// observers which consume at different rates.
	// defaultSubscribeBufferSize is the buffer size of a channelObserver's channel.
	defaultSubscribeBufferSize = 50
	// sendRetryInterval is the duration between attempts to send on the observer's
	// channel in the event that it's full. It facilitates a branch in a for loop
	// which unlocks the observer's mutex and tries again.
	// NOTE: setting this too low can cause the send retry loop to "slip", giving
	// up on a send attempt before the channel is ready to receive for multiple
	// iterations of the loop.
	sendRetryInterval = 100 * time.Millisecond
)

var _ observable.Observer[any] = (*channelObserver[any])(nil)

// channelObserver implements the observable.Observer interface.
type channelObserver[V any] struct {
	ctx context.Context
	// onUnsubscribe is called in Observer#Unsubscribe, closing this observer's
	// channel and removing it from the respective obervable's observers list
	// in a concurrency-safe manner.
	onUnsubscribe func(toRemove observable.Observer[V])
	// observerMu protects the observerCh and isClosed fields.
	observerMu *sync.RWMutex
	// observerCh is the channel that is used to emit values to the observer.
	// I.e. on the "N" side of the 1:N relationship between observable and
	// observer.
	observerCh chan V
	// isClosed indicates whether the observer has been isClosed. It's set in
	// unsubscribe; isClosed observers can't be reused.
	isClosed bool
}

type UnsubscribeFunc[V any] func(toRemove observable.Observer[V])

func NewObserver[V any](
	ctx context.Context,
	onUnsubscribe UnsubscribeFunc[V],
) *channelObserver[V] {
	// Create a channel for the observer and append it to the observers list
	return &channelObserver[V]{
		ctx:           ctx,
		observerMu:    new(sync.RWMutex),
		observerCh:    make(chan V, defaultSubscribeBufferSize),
		onUnsubscribe: onUnsubscribe,
	}
}

// Unsubscribe closes the subscription channel and removes the subscription from
// the observable.
func (obsvr *channelObserver[V]) Unsubscribe() {
	obsvr.unsubscribe()
}

// Ch returns a receive-only subscription channel.
func (obsvr *channelObserver[V]) Ch() <-chan V {
	return obsvr.observerCh
}

// IsClosed returns true if the observer has been unsubscribed.
// A closed observer cannot be reused.
func (obsvr *channelObserver[V]) IsClosed() bool {
	obsvr.observerMu.Lock()
	defer obsvr.observerMu.Unlock()

	return obsvr.isClosed
}

// unsubscribe closes the subscription channel, marks the observer as isClosed, and
// removes the subscription from its observable's observers list via onUnsubscribe.
func (obsvr *channelObserver[V]) unsubscribe() {
	obsvr.observerMu.Lock()
	defer obsvr.observerMu.Unlock()

	if obsvr.isClosed {
		// Get a context, eihter from the observer or from the background to get
		// a reference to the logger.
		ctx := obsvr.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		logger := polylog.Ctx(ctx)

		// log the fact that this case was encountered such that an extreme change
		// in its frequency would be obvious.
		logger.Warn().Err(observable.ErrObserverClosed).Msg("redundant unsubscribe")
		return
	}

	close(obsvr.observerCh)
	obsvr.isClosed = true
	obsvr.onUnsubscribe(obsvr)
}

// notify is called by observable to send a msg on the observer's channel.
// We can't use channelObserver#Ch because it's intended to be a
// receive-only channel. The channel will block if it is full (determined by the buffer
// size)
// if the channel's buffer is full, we will retry after sendRetryInterval/s.
// The other half is spent holding the read-lock and waiting for the (full) channel
// to be ready to receive.
func (obsvr *channelObserver[V]) notify(value V) {
	defer obsvr.observerMu.RUnlock() // defer releasing a read lock

	sendRetryTicker := time.NewTicker(sendRetryInterval)
	for {
		// observerMu must remain read-locked until the value is sent on observerCh
		// in the event that it would be isClosed concurrently (i.e. this observer
		// unsubscribes), which could cause a "send on isClosed channel" error.
		if !obsvr.observerMu.TryRLock() {
			continue
		}
		if obsvr.isClosed {
			return
		}

		select {
		case <-obsvr.ctx.Done():
			// if the context is done just release the read-lock (deferred)
			return
		case obsvr.observerCh <- value:
			// if observerCh has space in its buffer, the value is written to it
			return
		// if the context isn't done and channel is full (i.e. blocking),
		// release the read-lock to give write-lockers a turn. This case
		// continues the loop, re-read-locking and trying again.
		case <-sendRetryTicker.C:
			// TODO_IMPROVE: this is where we would implement
			// some backpressure strategy. It would be good to have a simple fail-
			// safe strategy that can be used by default; e.g. dropping the oldest
			// value if its buffer is full.

			// This case implies that the (read) lock was acquired, so it must
			// be unlocked before continuing the send retry loop.
			obsvr.observerMu.RUnlock()
		}
	}
}
