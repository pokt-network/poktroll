//go:build integration

package comet_query

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCometQueryClient_EventsObservable(t *testing.T) {
	var (
		newBlockQuery        = "tm.event='NewBlock'"
		notifyTimeout        = 5 * time.Second
		observedEventCounter = new(uint64)
		ctx                  = context.Background()
	)

	cClient, err := NewCometQueryClient("tcp://localhost:36657", "/websocket")
	require.NoError(t, err)
	require.NotNil(t, cClient)

	eventsObservable, err := cClient.EventsBytes(ctx, newBlockQuery)
	require.NoError(t, err)

	testCtx, testDone := context.WithCancel(context.Background())
	eventsObserver := eventsObservable.Subscribe(ctx)
	go func() {
		for eitherEvent := range eventsObserver.Ch() {
			event, err := eitherEvent.ValueOrError()
			assert.NoError(t, err)

			t.Log("received event:")
			t.Logf("%s", event)
			atomic.AddUint64(observedEventCounter, 1)
			testDone()
		}
	}()

	select {
	case <-testCtx.Done():
		require.ErrorIs(t, ctx.Err(), context.Canceled)
	case <-time.After(notifyTimeout):
		t.Fatal("timeout waiting for error channel to receive")
	}
}
