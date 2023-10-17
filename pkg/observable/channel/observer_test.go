package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestObserver_Unsubscribe(t *testing.T) {
	var (
		onUnsubscribeCalled = false
		inputCh             = make(chan int, 1)
	)
	obsvr := &channelObserver[int]{
		observerMu: &sync.RWMutex{},
		// using a buffered  channel to keep the test synchronous
		observerCh: inputCh,
		onUnsubscribe: func(toRemove *channelObserver[int]) {
			onUnsubscribeCalled = true
		},
	}

	// should initially be open
	require.Equal(t, false, obsvr.closed)

	inputCh <- 1
	require.Equal(t, false, obsvr.closed)

	obsvr.Unsubscribe()
	// should be closed after `#Unsubscribe()`
	require.Equal(t, true, obsvr.closed)
	require.True(t, onUnsubscribeCalled)
}

func TestObserver_ConcurrentUnsubscribe(t *testing.T) {
	var (
		onUnsubscribeCalled = false
		inputCh             = make(chan int, 1)
	)
	obsvr := &channelObserver[int]{
		ctx:        context.Background(),
		observerMu: &sync.RWMutex{},
		// using a buffered  channel to keep the test synchronous
		observerCh: inputCh,
		onUnsubscribe: func(toRemove *channelObserver[int]) {
			onUnsubscribeCalled = true
		},
	}

	// should initially be open
	require.Equal(t, false, obsvr.closed)

	done := make(chan struct{}, 1)
	go func() {
		for inputIdx := 0; ; inputIdx++ {
			select {
			case <-done:
				return
			default:
			}

			obsvr.notify(inputIdx)
			//time.Sleep(50 * time.Millisecond)
		}
	}()
	t.Cleanup(func() { done <- struct{}{} })

	// wait a bit, then assert that the observer is still open
	time.Sleep(50 * time.Millisecond)

	require.Equal(t, false, obsvr.closed)

	obsvr.Unsubscribe()
	// should be closed after `#Unsubscribe()`
	require.Equal(t, true, obsvr.closed)
	require.True(t, onUnsubscribeCalled)
}
