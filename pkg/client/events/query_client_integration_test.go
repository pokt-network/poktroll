//go:build integration

package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/testclient/testeventsquery"
)

// The query use to subscribe for new block events on the websocket endpoint exposed by CometBFT nodes
const committedBlockEventsQuery = "tm.event='NewBlock'"

func TestQueryClient_EventsObservable_Integration(t *testing.T) {
	const (
		eventReceiveTimeout = 5 * time.Second
		observedEventsLimit = 3
	)
	ctx := context.Background()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	// Start a subscription to the committed block events query. This begins
	// publishing events to the returned observable.
	eventsObservable, err := queryClient.EventsBytes(ctx, committedBlockEventsQuery)
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
