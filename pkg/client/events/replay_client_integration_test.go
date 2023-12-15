package events_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

var _ messageEvent = (*tEvent)(nil)

type messageEvent interface {
	EventMessage() string
}

type tEvent struct {
	Message string `json:"message"`
}

type messageEventReplayObs observable.ReplayObservable[messageEvent]

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

func newMessageEventBz(eventNum int32) []byte {
	return []byte(fmt.Sprintf(`{"message":"message_%d"}`, eventNum))
}

func TestReplayClient_Remapping(t *testing.T) {
	var (
		ctx              = context.Background()
		connClosed       atomic.Bool
		delayEvent       atomic.Bool
		readEventCounter int
		eventsReceived   atomic.Int32
		eventsToRecv     = int32(10)
		errCh            = make(chan error, 1)
		timeoutAfter     = 3 * time.Second // 1 second delay on retry.OnError
	)

	// Setup the mock connection and dialer
	ctrl := gomock.NewController(t)
	connMock := mockclient.NewMockConnection(ctrl)
	dialerMock := mockclient.NewMockDialer(ctrl)
	// Expect the connection to be closed and the dialer to be re-established
	connMock.EXPECT().Close().DoAndReturn(func() error {
		t.Logf("closing connection")
		connClosed.CompareAndSwap(false, true)
		return nil
	}).AnyTimes()
	// Expect the subscription to be re-established any number of times
	connMock.EXPECT().
		Send(gomock.Any()).
		DoAndReturn(func(eventBz []byte) error {
			t.Log("connecting")
			if connClosed.Load() {
				t.Log("opening connection")
				connClosed.CompareAndSwap(true, false)
			}
			t.Log("delaying next event")
			delayEvent.CompareAndSwap(true, false)
			return nil
		}).
		AnyTimes()
	// Mock the connection receiving events
	// TODO_IN_THIS_PR: Why do the calls after reconncetion not get received
	// by the replay client?
	connMock.EXPECT().Receive().
		DoAndReturn(func() (any, error) {
			// Simulate ErrConnClosed if connection is isClosed.
			if connClosed.Load() {
				t.Log("connection closed")
				return nil, events.ErrEventsConnClosed
			}

			// Delay the event if needed
			if !delayEvent.Load() {
				t.Log("delaying event")
				time.Sleep(50 * time.Millisecond)
				delayEvent.CompareAndSwap(false, true)
			}

			event := newMessageEventBz(int32(readEventCounter))
			readEventCounter++

			// Simulate IO delay between sequential events.
			time.Sleep(50 * time.Microsecond)

			return event, nil
		}).
		MinTimes(int(eventsToRecv))
	// Expect the dialer to be re-established any number of times
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (client.Connection, error) {
			t.Log("dialing connection")
			return connMock, nil
		}).
		AnyTimes()

	// Setup the events query client dependency
	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("", dialerOpt)
	deps := depinject.Supply(queryClient)

	// Create the replay client
	replayClient, err := events.NewEventsReplayClient[messageEvent, messageEventReplayObs](
		ctx,
		deps,
		"", // subscription query string
		newDecodeEventMessageFn(),
		100, // replay buffer size
	)
	require.NoError(t, err)

	go func() {
		// Subscribe to the replay clients events
		replayObs := replayClient.EventsSequence(ctx)
		replaySub := replayObs.Subscribe(ctx)
		var previousMessage messageEvent
		for msgEvent := range replaySub.Ch() {
			t.Logf("received event: %s", msgEvent.EventMessage())
			var previousNum int32
			var currentNum int32
			if previousMessage != nil {
				_, err := fmt.Sscanf(previousMessage.EventMessage(), "message_%d", &previousNum)
				require.NoError(t, err)
				_, err = fmt.Sscanf(msgEvent.EventMessage(), "message_%d", &currentNum)
				require.NoError(t, err)
				if !assert.Equal(t, previousNum+1, currentNum) {
					errCh <- fmt.Errorf("expected message number %d, got %d", previousNum+1, currentNum)
					return
				}
			}
			previousMessage = msgEvent

			require.NotEmpty(t, msgEvent)
			received := eventsReceived.Add(1)
			if received >= eventsToRecv {
				errCh <- nil
				return
			}
		}
	}()

	// Wait for ~2 events to be received
	time.Sleep(51600 * time.Microsecond)
	// Close the connection
	connMock.Close()

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equalf(t, eventsToRecv, eventsReceived.Load(), "received %d events, want: %d", eventsReceived.Load(), eventsToRecv)
	case <-time.After(timeoutAfter):
		t.Fatalf(
			"timed out waiting for events subscription; expected %d messages, got %d",
			eventsToRecv, eventsReceived.Load(),
		)
	}
}
