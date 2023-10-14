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
		observedEventsLimit = 2
	)
	ctx := context.Background()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	eventsObservable, errCh := queryClient.EventsBytes(ctx, "tm.event='NewBlock'")
	// check for a synchronous error
	// TECHDEBT(#70): once Either is available, we can remove the error channel and
	// instead use an Observable[Either[any, error]]. Return signature can become
	// `(obsvbl Observable[Either[[]byte, error]], syncErr error)`, where syncErr
	// is used for synchronous errors and obsvbl will propagate async errors via
	// the Either.
	select {
	case err := <-errCh:
		require.NoError(t, err)
	default:
		// no synchronous error
	}
	eventsObserver := eventsObservable.Subscribe(ctx)

	var eventCounter int
	go func() {
		for range eventsObserver.Ch() {
			eventCounter++

			if eventCounter >= observedEventsLimit {
				errCh <- nil
				return
			}
		}
	}()

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equal(t, observedEventsLimit, eventCounter)
	case <-time.After(eventReceiveTimeout):
		t.Fatalf(
			"timed out waiting for block subscription; expected %d blocks, got %d",
			observedEventsLimit, eventCounter,
		)
	}
}
