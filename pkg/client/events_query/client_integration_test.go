//go:build integration

package eventsquery_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"pocket/internal/testclient/testeventsquery"
)

func TestQueryClient_EventsObservable_Integration(t *testing.T) {
	const (
		eventReceiveTimeout = 5 * time.Second
		observedEventsLimit = 3
	)
	ctx := context.Background()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	eventsObservable, err := queryClient.EventsBytes(ctx, "tm.event='NewBlock'")
	require.NoError(t, err)

	eventsObserver := eventsObservable.Subscribe(ctx)

	var (
		eventCounter int
		done         = make(chan struct{}, 1)
	)
	go func() {
		for range eventsObserver.Ch() {
			eventCounter++

			if eventCounter >= observedEventsLimit {
				done <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-done:
		require.NoError(t, err)
		require.Equal(t, observedEventsLimit, eventCounter)
	case <-time.After(eventReceiveTimeout):
		t.Fatalf(
			"timed out waiting for block subscription; expected %d blocks, got %d",
			observedEventsLimit, eventCounter,
		)
	}
}
