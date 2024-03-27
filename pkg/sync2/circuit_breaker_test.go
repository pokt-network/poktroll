package sync2_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/sync2"
)

func TestCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	limit := uint(3)
	circuitBreaker := sync2.NewCircuitBreaker(limit)
	delay := 100 * time.Millisecond
	delaysComplete := new(atomic.Uint64)

	delayFn := func() {
		time.Sleep(delay)
		delaysComplete.Add(1)
	}

	// Circuit breaker should not trip.
	for i := uint(0); i < limit; i++ {
		ok := circuitBreaker.Go(ctx, delayFn)
		require.True(t, ok)
	}

	// Wait for the initial goroutines to finish.
	<-time.After(delay * time.Duration(limit+1))

	// Circuit breaker should trip.
	for i := uint(0); i < limit+1; i++ {
		ok := circuitBreaker.Go(ctx, delayFn)

		if i < limit {
			require.True(t, ok)
		} else {
			require.False(t, ok)
		}
	}

	circuitBreaker.Close()
	// Calling #Close() multiple times should not panic.
	circuitBreaker.Close()

	require.Equal(t, uint64(limit*2), delaysComplete.Load())

	ok := circuitBreaker.Go(ctx, delayFn)
	require.Falsef(t, ok, "circuit breaker should be closed")

	require.Equalf(t, uint64(limit*2), delaysComplete.Load(), "delaysComplete should not have changed")
}
