package events_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
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
	return func(eventResult *coretypes.ResultEvent) (messageEvent, error) {
		if data, ok := eventResult.Data.([]byte); ok {
			event := &tEvent{}
			err := json.Unmarshal(data, event)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal event message: %w", err)
			}
			return event, nil
		}
		return nil, fmt.Errorf("failed to decode event message")
	}
}

// newMessageEventBz returns a new message event in JSON format
func newMessageEventBz(eventNum int32) []byte {
	return []byte(fmt.Sprintf(`{"message":"message_%d"}`, eventNum))
}

func TestReplayClient_Remapping(t *testing.T) {
	var (
		ctx               = context.Background()
		firstEventDelayed atomic.Bool
		readEventCounter  atomic.Int32
		eventsReceived    atomic.Int32
		eventsToRecv      = int32(10)
		errCh             = make(chan error, 1)
		timeoutAfter      = 3 * time.Second // 1 second delay on retry.OnError
	)

	resultEventCh := make(chan coretypes.ResultEvent, 1)

	go func() {
		for range eventsToRecv {
			if !firstEventDelayed.CompareAndSwap(false, true) {
				time.Sleep(50 * time.Millisecond)
			}

			eventNum := readEventCounter.Add(1) - 1
			event := coretypes.ResultEvent{
				Data: newMessageEventBz(eventNum),
			}

			resultEventCh <- event
			time.Sleep(50 * time.Microsecond)
		}
	}()

	ctrl := gomock.NewController(t)
	cometHTTPClientMock := mockclient.NewMockClient(ctrl)
	cometHTTPClientMock.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(resultEventCh, nil).
		Times(1)

	logger := polylog.Ctx(ctx)
	deps := depinject.Supply(cometHTTPClientMock, logger)

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
