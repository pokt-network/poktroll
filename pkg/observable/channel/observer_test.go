package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable"
)

func TestObserver_Unsubscribe(t *testing.T) {
	var (
		publishCh           = make(chan int, 1)
		onUnsubscribeCalled = false
		onUnsubscribe       = func(toRemove observable.Observer[int]) {
			onUnsubscribeCalled = true
		}
	)
	obsvr := &channelObserver[int]{
		observerMu: &sync.RWMutex{},
		// using a buffered channel to keep the test synchronous
		observerCh:    publishCh,
		onUnsubscribe: onUnsubscribe,
	}

	// should initially be open
	require.Equal(t, false, obsvr.isClosed)

	publishCh <- 1
	require.Equal(t, false, obsvr.isClosed)

	obsvr.Unsubscribe()
	// should be isClosed after `#Unsubscribe()`
	require.Equal(t, true, obsvr.isClosed)
	require.True(t, onUnsubscribeCalled)
}

func TestObserver_ConcurrentUnsubscribe(t *testing.T) {
	var (
		publishCh           = make(chan int, 1)
		onUnsubscribeCalled = false
		onUnsubscribe       = func(toRemove observable.Observer[int]) {
			onUnsubscribeCalled = true
		}
	)

	obsvr := &channelObserver[int]{
		ctx:        context.Background(),
		observerMu: &sync.RWMutex{},
		// using a buffered channel to keep the test synchronous
		observerCh:    publishCh,
		onUnsubscribe: onUnsubscribe,
	}

	require.Equal(t, false, obsvr.isClosed, "observer channel should initially be open")

	// concurrently & continuously publish until the test cleanup runs
	done := make(chan struct{}, 1)
	go func() {
		for idx := 0; ; idx++ {
			// return when done receives; otherwise,
			select {
			case <-done:
				return
			default:
			}

			// publish a value
			obsvr.notify(idx)

			// Slow this loop to prevent bogging the test down.
			time.Sleep(10 * time.Microsecond)
		}
	}()
	// send on done when the test cleans up
	t.Cleanup(func() { done <- struct{}{} })

	// it should still be open after a bit of inactivity
	time.Sleep(time.Millisecond)
	require.Equal(t, false, obsvr.isClosed)

	obsvr.Unsubscribe()
	// should be isClosed after `#Unsubscribe()`
	require.Equal(t, true, obsvr.isClosed)
	require.True(t, onUnsubscribeCalled)
}
