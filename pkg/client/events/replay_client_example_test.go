package events_test

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/pocket/pkg/client/events"
	"github.com/pokt-network/pocket/pkg/polylog"
)

const (
	// Define a query string to provide to the EventsQueryClient
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	// And: https://docs.cosmos.network/v0.47/learn/advanced/events#default-events
	eventQueryString = "message.action='messageActionName'"
	// Define the websocket URL the EventsQueryClient will subscribe to
	cometWebsocketURL = "ws://example.com:26657/websocket"
	// the amount of events we want before they are emitted
	replayObsBufferSize = 1
)

var _ EventType = (*eventType)(nil)

// Define an interface to represent the onchain event
type EventType interface {
	GetName() string // Illustrative only; arbitrary interfaces are supported.
}

// Define the event type that implements the interface
type eventType struct {
	Name string `json:"name"`
}

// See: https://pkg.go.dev/github.com/pokt-network/pocket/pkg/client/events/#NewEventsFn
func eventTypeFactory(ctx context.Context) events.NewEventsFn[EventType] {
	// Define a decoder function that can take the raw event bytes
	// received from the EventsQueryClient and convert them into
	// the desired type for the EventsReplayClient
	return func(eventBz []byte) (EventType, error) {
		eventMsg := new(eventType)
		logger := polylog.Ctx(ctx)

		if err := json.Unmarshal(eventBz, eventMsg); err != nil {
			return nil, err
		}

		// Confirm the event is correct by checking its fields
		if eventMsg.Name == "" {
			logger.Error().Str("eventBz", string(eventBz)).Msg("event type is not correct")
			return nil, events.ErrEventsUnmarshalEvent.
				Wrapf("with eventType data: %s", string(eventBz))
		}

		return eventMsg, nil
	}
}

func (e *eventType) GetName() string { return e.Name }

func ExampleNewEventsReplayClient() {
	// Create the events query client and a depinject config to supply
	// it into the EventsReplayClient
	// See: https://pkg.go.dev/github.com/pokt-network/pocket/pkg/client/events/#NewEventsQueryClient
	evtClient := events.NewEventsQueryClient(cometWebsocketURL)
	depConfig := depinject.Supply(evtClient)

	// Create a context (this should be cancellable to close the EventsReplayClient)
	ctx, cancel := context.WithCancel(context.Background())

	// Create a new instance of the EventsReplayClient
	// See: https://pkg.go.dev/github.com/pokt-network/pocket/pkg/client/events/#NewEventsReplayClient
	client, err := events.NewEventsReplayClient[EventType](
		ctx,
		depConfig,
		eventQueryString,
		eventTypeFactory(ctx),
		replayObsBufferSize,
	)
	if err != nil {
		panic(fmt.Errorf("unable to create EventsReplayClient %v", err))
	}

	// Assume events the lastest event emitted of type EventType has the name "testEvent"

	// Retrieve the latest emitted event
	lastEventType := client.LastNEvents(ctx, 1)[0]
	fmt.Printf("Last Event: '%s'\n", lastEventType.GetName())

	// Get the latest replay observable from the EventsReplayClient
	// In order to get the latest events from the sequence
	latestEventsObs := client.EventsSequence(ctx)
	// Get the latest events from the sequence
	lastEventType = latestEventsObs.Last(ctx, 1)[0]
	fmt.Printf("Last Event: '%s'\n", lastEventType.GetName())

	// Cancel the context which will call client.Close and close all
	// subscriptions and the EventsQueryClient
	cancel()
	// Output
	// Last Event: 'testEvent'
	// Last Event: 'testEvent'
}
