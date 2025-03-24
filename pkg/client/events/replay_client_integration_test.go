package events_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

// Create the generic event type and decoder for the replay client

var _ messageEvent = (*tEvent)(nil)

type messageEvent interface {
	EventMessage() string
}

type tEvent struct {
	Message string `json:"message"`
}

func (t *tEvent) EventMessage() string {
	return t.Message
}

func newDecodeEventMessageFn() events.NewEventsFn[messageEvent] {
	return func(eventBz []byte) (messageEvent, error) {
		t := new(tEvent)
		if err := json.Unmarshal(eventBz, t); err != nil {
			return nil, err
		}
		if t.Message == "" {
			return nil, events.ErrEventsUnmarshalEvent
		}
		return t, nil
	}
}

// newMessageEventBz returns a new message event in JSON format
func newMessageEventBz(eventNum int32) []byte {
	return []byte(fmt.Sprintf(`{"message":"message_%d"}`, eventNum))
}

func TestReplayClient_Remapping(t *testing.T) {
	var (
		ctx               = context.Background()
		connClosed        atomic.Bool
		firstEventDelayed atomic.Bool
		readEventCounter  atomic.Int32
		eventsReceived    atomic.Int32
		eventsToRecv      = int32(10)
		errCh             = make(chan error, 1)
		timeoutAfter      = 3 * time.Second // 1 second delay on retry.OnError
	)

	// Setup the mock connection and dialer
	connMock, dialerMock := testeventsquery.NewNTimesReconnectMockConnAndDialer(t, 2, &connClosed, &firstEventDelayed)
	// Mock the connection receiving events
	connMock.EXPECT().Receive().
		// Receive is called in the tightest loop possible (max speed limited
		// by a goroutine) and as such the sleep's within are used to slow down
		// the time between events to prevent unexpected behavior. As in this
		// test environment, there are no "real" delays between "#Receive" calls
		// (events being emitted) and as such the sleep's enable the publishing
		// of notifications to observers to occur in a flake-free manner.
		DoAndReturn(func() (any, error) {
			// Simulate ErrConnClosed if connection is isClosed.
			if connClosed.Load() {
				return nil, events.ErrEventsConnClosed
			}

			// Delay the event if needed, this is to allow for the events query
			// client to subscribe and receive the first event.
			if !firstEventDelayed.CompareAndSwap(false, true) {
				time.Sleep(50 * time.Millisecond)
			}

			eventNum := readEventCounter.Add(1) - 1
			event := newMessageEventBz(eventNum)
			// After an arbitrary number of events (2 in this case), simulate
			// the connection closing so that the replay client can remap the
			// events it receives without the caller having to resubscribe.
			if eventNum == 2 {
				// Simulate the connection closing
				connMock.Close()
			}

			// Simulate IO delay between sequential events.
			time.Sleep(50 * time.Microsecond)

			return event, nil
		}).
		MinTimes(int(eventsToRecv))

	// Setup the events query client dependency
	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("", dialerOpt)
	deps := depinject.Supply(queryClient)

	// Create the replay client
	replayClient, err := events.NewEventsReplayClient[messageEvent](
		ctx,
		deps,
		"", // subscription query string
		newDecodeEventMessageFn(),
		100, // replay buffer size
	)
	require.NoError(t, err)

	channel.ForEach(
		ctx,
		observable.Observable[messageEvent](replayClient.EventsSequence(ctx)),
		func(ctx context.Context, event messageEvent) {
			require.NotEmpty(t, event)
			received := eventsReceived.Add(1)
			if received >= eventsToRecv {
				errCh <- nil
				return
			}
		},
	)

	select {
	case err := <-errCh:
		require.NoError(t, err)
		eventsRecv := eventsReceived.Load()
		require.Equalf(t, eventsToRecv, eventsRecv, "received %d events, want: %d", eventsReceived.Load(), eventsRecv)
	case <-time.After(timeoutAfter):
		t.Fatalf(
			"timed out waiting for events subscription; expected %d messages, got %d",
			eventsToRecv, eventsReceived.Load(),
		)
	}
}
