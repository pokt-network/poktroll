package events

import (
	"context"

	"cosmossdk.io/depinject"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// subscriptionReplayClient is the name of the subscription client used to subscribe
// to events via the CometBFT WebSocket connection.
const subscriptionReplayClient = "replay-client"

// Enforce the EventsReplayClient interface is implemented by the replayClient type.
var _ client.EventsReplayClient[any] = (*replayClient[any])(nil)

// NewEventsFn is a function that converts a ResultEvent into a new generic type T instance
type NewEventsFn[T any] func(*coretypes.ResultEvent) (T, error)

// replayClient:
// - Implements the EventsReplayClient interface for a generic type T
// - Provides a replay observable for type T

type replayClient[T any] struct {
	logger polylog.Logger

	// queryString:
	// - Query string used to subscribe to events of the desired type
	// - See: https://docs.cosmos.network/main/learn/advanced/events#subscribing-to-events
	// - See: https://docs.cosmos.network/main/learn/advanced/events#default-events
	queryString string

	// cometClient:
	// - CometBFT client used to subscribe to events via the WebSocket connection
	// - Provides direct access to the node's RPC endpoints for event subscription
	cometClient cometclient.Client

	// eventDecoder:
	// - Function which decodes event subscription into the type defined by the EventsReplayClient's generic type parameter
	eventDecoder NewEventsFn[T]

	// replayObsBufferSize:
	// - Buffer size for the replay observable returned by EventsSequence
	// - Can be any integer; refers to the number of notifications the replay observable will hold in its buffer
	// - Notifications can be replayed to new observers
	// - NB: This is not the buffer size of the replayObsCache
	replayObsBufferSize int

	// eventTypeObs is the replay observable for the generic type T.
	eventTypeObs observable.ReplayObservable[T]

	// replayEventTypeObsCh is the channel used to publish events of type T
	replayEventTypeObsCh chan<- T
}

// NewEventsReplayClient creates a new EventsReplayClient from the given dependencies
// and subscription query string.
//   - It requires a decoder function to be provided which decodes event subscription
//     result into the type defined by the EventsReplayClient's generic type parameter.
//   - The replayObsBufferSize is the replay buffer size of the replay observable
//     which is notified of new events.
//
// Required dependencies:
//   - cometClient: cometbft/rpc/client/http.HTTP
func NewEventsReplayClient[T any](
	ctx context.Context,
	deps depinject.Config,
	queryString string,
	newEventFn NewEventsFn[T],
	replayObsBufferSize int,
) (client.EventsReplayClient[T], error) {

	// Initialize the replay client
	rClient := &replayClient[T]{
		queryString:         queryString,
		eventDecoder:        newEventFn,
		replayObsBufferSize: replayObsBufferSize,
	}

	// Inject dependencies
	if err := depinject.Inject(deps, &rClient.cometClient, &rClient.logger); err != nil {
		return nil, err
	}

	// Create a new replay observable and publish channel for event type T with
	// a buffer size matching that provided during the EventsReplayClient
	// construction.
	eventTypeObs, replayEventTypeObsCh := channel.NewReplayObservable[T](
		ctx,
		rClient.replayObsBufferSize,
	)

	resultEventCh, err := rClient.cometClient.Subscribe(ctx, subscriptionReplayClient, rClient.queryString)
	if err != nil {
		return nil, err
	}

	// Store the event type observable.
	rClient.eventTypeObs = eventTypeObs
	rClient.replayEventTypeObsCh = replayEventTypeObsCh

	// Concurrently publish events to the observable emitted by replayEventTypeObsCh.
	go rClient.goPublishEvents(resultEventCh)

	return rClient, nil
}

// EventsSequence returns a new ReplayObservable, with the buffer size provided
// during the EventsReplayClient construction, which is notified when new
// events are received by the encapsulated EventsQueryClient.
func (rClient *replayClient[T]) EventsSequence(ctx context.Context) observable.ReplayObservable[T] {
	return rClient.eventTypeObs
}

// LastNEvents returns the last N typed events that have been received by the
// corresponding events query subscription.
// It blocks until at least one event has been received.
func (rClient *replayClient[T]) LastNEvents(ctx context.Context, n int) []T {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return rClient.EventsSequence(ctx).Last(ctx, n)
}

// goPublishEvents is a goroutine that listens for new events from the CometBFT
// subscription and publishes them to the replay observable.
func (rClient *replayClient[T]) goPublishEvents(resultEventCh <-chan coretypes.ResultEvent) {
	go func() {
		// Process events until connection breaks or context is canceled
		for resultEvent := range resultEventCh {
			// Attempt to decode the raw event bytes into the target type T
			event, err := rClient.eventDecoder(&resultEvent)
			if err != nil {
				rClient.logger.Error().Err(err).Msgf("❌ Event decoding failed! 🔄 Skipping and moving to the next event.")
				continue
			}

			rClient.replayEventTypeObsCh <- event
		}
	}()
}
