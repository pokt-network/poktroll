package events_test

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

const (
	// Define a query string to provide to the EventsQueryClient
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	// And: https://docs.cosmos.network/v0.47/learn/advanced/events#default-events
	eventQueryString = "message.action='messageActionName'"
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

// See: https://pkg.go.dev/github.com/pokt-network/poktroll/pkg/client/events/#NewEventsFn
func eventTypeFactory(ctx context.Context) events.NewEventsFn[EventType] {
	// Define a decoder function that can take the raw event bytes
	// received from the EventsQueryClient and convert them into
	// the desired type for the EventsReplayClient
	return func(eventResult *coretypes.ResultEvent) (EventType, error) {
		eventMsg, ok := eventResult.Data.(eventType)
		if !ok {
			return nil, fmt.Errorf("unable to decode event data: %T", eventResult.Data)
		}

		return &eventMsg, nil
	}
}

func (e *eventType) GetName() string { return e.Name }

func ExampleNewEventsReplayClient() {
	depConfig := depinject.Supply()

	// Create a context (this should be cancellable to close the EventsReplayClient)
	ctx, cancel := context.WithCancel(context.Background())

	// Create a new instance of the EventsReplayClient
	// See: https://pkg.go.dev/github.com/pokt-network/poktroll/pkg/client/events/#NewEventsReplayClient
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
