package events

import (
	"context"
	"math"
	"time"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	// DefaultConnRetryLimit is used to indicate how many times the
	// underlying replay client should attempt to retry if it encounters an error
	// or its connection is interrupted.
	//
	// TODO_IMPROVE: this should be configurable but can be overridden at compile-time:
	// go build -ldflags "-X github.com/pokt-network/poktroll/DefaultConnRetryLimit=value".
	// This is set to max int because the websocket client should always keep trying to reconnect.
	// Note that this parameter is only used by the websockets client.
	DefaultConnRetryLimit = math.MaxInt

	// eventsBytesRetryDelay is the delay between retry attempts when the events
	// bytes observable returns an error.
	eventsBytesRetryDelay = time.Second

	// TODO_MAINNET(@bryanchriswhite): Make this a customizable parameter in the
	// Gateway & RelayMiner config files
	// eventsBytesRetryLimit is the maximum number of times to attempt to
	// re-establish the events query bytes subscription when the events bytes
	// observable returns an error or closes.
	eventsBytesRetryLimit        = 10
	eventsBytesRetryResetTimeout = 10 * time.Second

	// replayObsCacheBufferSize is the replay buffer size of the
	// replayObsCache replay observable which is used to cache the replay
	// observable that is notified of new events.
	// It, replayObsCache, is updated with a new "active" observable when a new
	// events query subscription is created, for example, after a non-persistent
	// connection error.
	replayObsCacheBufferSize = 1
)

// Enforce the EventsReplayClient interface is implemented by the replayClient type.
var _ client.EventsReplayClient[any] = (*replayClient[any])(nil)

// NewEventsFn is a function that takes a byte slice and returns a new instance
// of the generic type T.
type NewEventsFn[T any] func([]byte) (T, error)

// replayClient implements the EventsReplayClient interface for a generic type T,
// and replay observable for type T.
type replayClient[T any] struct {
	// queryString is the query string used to subscribe to events of the
	// desired type.
	// See: https://docs.cosmos.network/main/learn/advanced/events#subscribing-to-events
	// and: https://docs.cosmos.network/main/learn/advanced/events#default-events
	queryString string
	// eventsClient is the events query client which is used to subscribe to
	// newly committed block events. It emits an either value which may contain
	// an error, at most, once and closes immediately after if it does.
	eventsClient client.EventsQueryClient
	// eventDecoder is a function which decodes event subscription
	// message bytes into the type defined by the EventsReplayClient's generic type
	// parameter.
	eventDecoder NewEventsFn[T]
	// replayObsBufferSize is the buffer size for the replay observable returned
	// by EventsSequence, this can be any integer and it refers to the number of
	// notifications the replay observable will hold in its buffer, that can be
	// replayed to new observers.
	// NB: This is not the buffer size of the replayObsCache
	replayObsBufferSize int
	// replayObsCache is a replay observable with a buffer size of 1, which
	// holds the "active latest replay observable" which is notified when new
	// events are received by the events query client subscription created in
	// goPublishEvents. This observable (and the one it emits) closes when the
	// events bytes observable returns an error and is updated with a new
	// "active" observable after a new events query subscription is created.
	//
	// TODO_TECHDEBT(@bryanchriswhite): Look into making this a regular observable as
	// we may no longer depend on it being replayable.
	replayObsCache observable.ReplayObservable[observable.ReplayObservable[T]]
	// replayObsCachePublishCh is the publish channel for replayObsCache.
	// It's used to set and subsequently update replayObsCache the events replay
	// observable;
	// For example when the connection is re-established after erroring.
	replayObsCachePublishCh chan<- observable.ReplayObservable[T]
	// eventTypeObs is the replay observable for the generic type T.
	eventTypeObs observable.ReplayObservable[T]
	// replayClientCancelCtx is the function to cancel the context of the replay client.
	// It is called when the replay client is closed.
	replayClientCancelCtx func()
	// connRetryLimit is the number of times the replay client should retry
	// in the event that it encounters an error or its connection is interrupted.
	// If connRetryLimit is < 0, it will retry indefinitely.
	connRetryLimit int

	logger polylog.Logger
}

// NewEventsReplayClient creates a new EventsReplayClient from the given
// dependencies, cometWebsocketURL and subscription query string. It requires a
// decoder function to be provided which decodes event subscription message
// bytes into the type defined by the EventsReplayClient's generic type parameter.
// The replayObsBufferSize is the replay buffer size of the replay observable
// which is notified of new events.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewEventsReplayClient[T any](
	ctx context.Context,
	deps depinject.Config,
	queryString string,
	newEventFn NewEventsFn[T],
	replayObsBufferSize int,
	opts ...client.EventsReplayClientOption[T],
) (client.EventsReplayClient[T], error) {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize the replay client
	rClient := &replayClient[T]{
		queryString:           queryString,
		eventDecoder:          newEventFn,
		replayObsBufferSize:   replayObsBufferSize,
		replayClientCancelCtx: cancel,
		connRetryLimit:        DefaultConnRetryLimit,
	}

	for _, opt := range opts {
		opt(rClient)
	}

	// TODO_TECHDEBT(@bryanchriswhite): Look into making this a regular observable as
	// we may no longer depend on it being replayable.
	replayObsCache, replayObsCachePublishCh := channel.NewReplayObservable[observable.ReplayObservable[T]](
		ctx,
		// Buffer size of 1 as the cache only needs to hold the latest
		// active replay observable.
		replayObsCacheBufferSize,
	)
	rClient.replayObsCache = replayObsCache
	rClient.replayObsCachePublishCh = replayObsCachePublishCh

	// Inject dependencies
	if err := depinject.Inject(deps, &rClient.eventsClient, &rClient.logger); err != nil {
		return nil, err
	}

	// Create a new replay observable and publish channel for event type T with
	// a buffer size matching that provided during the EventsReplayClient
	// construction.
	eventTypeObs, replayEventTypeObsPublishCh := channel.NewReplayObservable[T](
		ctx,
		rClient.replayObsBufferSize,
	)

	// Concurrently publish events to the observable emitted by replayObsCache.
	go rClient.goPublishEvents(ctx, replayEventTypeObsPublishCh)

	// Store the event type observable.
	rClient.eventTypeObs = eventTypeObs

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

// Close unsubscribes all observers of the committed blocks sequence observable
// and closes the events query client.
func (rClient *replayClient[T]) Close() {
	// Closing eventsClient will cascade unsubscribe and close downstream observers.
	rClient.eventsClient.Close()
	// Close all the downstream observers of the replay client.
	rClient.replayClientCancelCtx()
}

// goPublishEvents establishes and maintains an events subscription by
// - Establishing EventsBytes subscription to events matching the query string
// - Processes incoming events by decoding them from bytes to the generic type T
// - Retries connection on failures up to connRetryLimit times with predefined delay
// - Handles context cancellation to prevent resource leaks
// - Cleans up connections when errors occur or context is canceled
//
// The method runs in a continuous loop until the context is cancelled or retry limit is exceeded.
func (rClient *replayClient[T]) goPublishEvents(ctx context.Context, publishCh chan<- T) {
	numRetries := 0

	for {
		// Check if retry limit has been exceeded
		if numRetries > rClient.connRetryLimit {
			// If the number of retries exceeds the connection retry limit, exit the loop.
			rClient.logger.Error().Msgf(
				"Exceeded connection retry limit of %d for events query subscription with query: %s",
				rClient.connRetryLimit,
				rClient.queryString,
			)
			return
		}

		select {
		case <-ctx.Done():
			// If the context is done, exit the loop and stop processing events.
			return
		default:
			// Create a cancellable context for this connection attempt
			eventsBzCtx, cancelEventsBzObs := context.WithCancel(ctx)

			// Attempt to establish an EventsBytes subscription
			// This will return an observable that emits either event bytes or an error
			eventsBytesObs, err := rClient.eventsClient.EventsBytes(eventsBzCtx, rClient.queryString)
			if err != nil {
				rClient.logger.Error().Err(err).Msgf(
					"Failed to establish events bytes subscription for query: %s, retrying (%d/%d)",
					rClient.queryString,
					numRetries+1,
					rClient.connRetryLimit,
				)

				// Connection failed - clean up and retry
				cancelEventsBzObs()
				numRetries++
				time.Sleep(eventsBytesRetryDelay)

				continue
			}

			// Subscribe to the events observable and get the channel for receiving events
			eventsCh := eventsBytesObs.Subscribe(eventsBzCtx).Ch()

			// Process events until connection breaks or context is cancelled
			for eitherEventBz := range eventsCh {
				// Extract event bytes or error from the Either type
				eventBz, eitherErr := eitherEventBz.ValueOrError()
				if eitherErr != nil {
					rClient.logger.Error().Err(eitherErr).Msgf(
						"Error receiving events bytes for query: %s, retrying (%d/%d)",
						rClient.queryString,
						numRetries+1,
						rClient.connRetryLimit,
					)

					// Connection error occurred - exit event loop to retry
					break
				}

				// Attempt to decode the raw event bytes into the target type T
				event, err := rClient.eventDecoder(eventBz)
				if err != nil {
					if ErrEventsUnmarshalEvent.Is(err) {
						// Event bytes were not the expected type - skip this event and continue
						continue
					}

					// Unexpected decoding error - exit event loop to retry
					break
				}

				// Successfully decoded event - publish it to the channel
				publishCh <- event
			}

			// Cancel the context to cleanup the subscription
			cancelEventsBzObs()

			// Increment retry counter and delay
			numRetries++
			time.Sleep(eventsBytesRetryDelay)
		}
	}
}
