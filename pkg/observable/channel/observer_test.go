package channel

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObserver_Unsubscribe(t *testing.T) {
	var (
		onUnsubscribeCalled = false
		inputCh             = make(chan int, 1)
	)
	sub := &channelObserver[int]{
		observerMu: &sync.RWMutex{},
		// using a buffered  channel to keep the test synchronous
		observerCh: inputCh,
		onUnsubscribe: func(toRemove *channelObserver[int]) {
			onUnsubscribeCalled = true
		},
	}

	// should initially be open
	require.Equal(t, false, sub.closed)

	inputCh <- 1
	require.Equal(t, false, sub.closed)

	sub.Unsubscribe()
	// should be closed after `#Unsubscribe()`
	require.Equal(t, true, sub.closed)
	require.True(t, onUnsubscribeCalled)
}
