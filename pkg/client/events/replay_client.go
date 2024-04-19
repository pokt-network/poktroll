package events

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/retry"
)

const (
	// eventsBytesRetryDelay is the delay between retry attempts when the events
	// bytes observable returns an error.
	eventsBytesRetryDelay = time.Second
	// eventsBytesRetryLimit is the maximum number of times to attempt to
	// re-establish the events query bytes subscription when the events bytes
	// observable returns an error or closes.
	eventsBytesRetryLimit        = 0
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
	// TODO_REFACTOR(@h5law): Look into making this a regular observable as
	// we may no longer depend on it being replayable.
	replayObsCache observable.ReplayObservable[observable.ReplayObservable[T]]
	// replayObsCachePublishCh is the publish channel for replayObsCache.
	// It's used to set and subsequently update replayObsCache the events replay
	// observable;
	// For example when the connection is re-established after erroring.
	replayObsCachePublishCh chan<- observable.ReplayObservable[T]
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
) (client.EventsReplayClient[T], error) {
	// Initialize the replay client
	rClient := &replayClient[T]{
		queryString:         queryString,
		eventDecoder:        newEventFn,
		replayObsBufferSize: replayObsBufferSize,
	}
	// TODO_REFACTOR(@h5law): Look into making this a regular observable as
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
	if err := depinject.Inject(deps, &rClient.eventsClient); err != nil {
		return nil, err
	}

	// Concurrently publish events to the observable emitted by replayObsCache.
	go rClient.goPublishEvents(ctx)

	return rClient, nil
}

// EventsSequence returns a new ReplayObservable, with the buffer size provided
// during the EventsReplayClient construction, which is notified when new
// events are received by the encapsulated EventsQueryClient.
func (rClient *replayClient[T]) EventsSequence(ctx context.Context) observable.ReplayObservable[T] {
	// Create a new replay observable and publish channel for event type T with
	// a buffer size matching that provided during the EventsReplayClient
	// construction.
	eventTypeObs, replayEventTypeObsPublishCh := channel.NewReplayObservable[T](
		ctx,
		rClient.replayObsBufferSize,
	)

	// Ensure that the subscribers of the returned eventTypeObs receive
	// notifications from the latest open replay observable.
	go rClient.goRemapEventsSequence(ctx, replayEventTypeObsPublishCh)

	// Return the event type observable.
	return eventTypeObs
}

// goRemapEventsSequence publishes events observed by the most recent cached
// events type replay observable to the given publishCh
func (rClient *replayClient[T]) goRemapEventsSequence(ctx context.Context, publishCh chan<- T) {
	channel.ForEach[observable.ReplayObservable[T]](
		ctx,
		rClient.replayObsCache,
		func(ctx context.Context, eventTypeObs observable.ReplayObservable[T]) {
			eventObserver := eventTypeObs.Subscribe(ctx)
			for event := range eventObserver.Ch() {
				publishCh <- event
			}
		},
	)
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
	close(rClient.replayObsCachePublishCh)
}

// goPublishEvents runs the work function returned by retryPublishEventsFactory,
// re-invoking it according to the arguments to retry.OnError when the events bytes
// observable returns an asynchronous error.
// This function is intended to be called in a goroutine.
func (rClient *replayClient[T]) goPublishEvents(ctx context.Context) {
	// React to errors by getting a new events bytes observable, re-mapping it,
	// and send it to replayObsCachePublishCh such that
	// replayObsCache.Last(ctx, 1) will return it.
	publishError := retry.OnError(
		ctx,
		eventsBytesRetryLimit,
		eventsBytesRetryDelay,
		eventsBytesRetryResetTimeout,
		"goPublishEvents",
		rClient.retryPublishEventsFactory(ctx),
	)

	// If we get here, the retry limit was reached and the retry loop exited.
	// Since this function runs in a goroutine, we can't return the error to the
	// caller. Instead, we panic.
	if publishError != nil {
		panic(fmt.Errorf("EventsReplayClient[%T].goPublishEvents should never reach this spot: %w", *new(T), publishError))
	}
}

// retryPublishEventsFactory returns a function which is intended to be passed
// to retry.OnError. The returned function pipes event bytes from the events
// query client, maps them to typed events, and publishes them to the
// replayObsCache replay observable.
func (rClient *replayClient[T]) retryPublishEventsFactory(ctx context.Context) func() chan error {
	return func() chan error {
		errCh := make(chan error, 1)
		eventsBytesObs, err := rClient.eventsClient.EventsBytes(ctx, rClient.queryString)
		if err != nil {
			errCh <- err
			return errCh
		}

		// NB: must cast back to generic observable type to use with Map.
		eventsBzObs := observable.Observable[either.Either[[]byte]](eventsBytesObs)
		typedObs := channel.MapReplay(
			ctx,
			replayObsCacheBufferSize,
			eventsBzObs,
			rClient.newMapEventsBytesToTFn(errCh),
		)

		// Subscribe to the eventBzObs and block until the channel closes.
		// Then pass this as an error to force the retry.OnError to resubscribe.
		go func() {
			eventsBzObserver := eventsBzObs.Subscribe(ctx)
			for range eventsBzObserver.Ch() {
				// Wait for the channel to close.
				continue
			}
			// UnsubscribeAll downstream observers, as the source observable has
			// closed and will not emit any more values.
			typedObs.UnsubscribeAll()
			// Publish an error to the error channel to initiate a retry
			errCh <- ErrEventsConsClosed
		}()

		// Initially set replayObsCache and update if after retrying on error.
		rClient.replayObsCachePublishCh <- typedObs

		return errCh
	}
}

// newMapEventsBytesToTFn is a factory for a function which is intended
// to be used as a transformFn in a channel.Map() call. Since the map function
// is called asynchronously, this factory creates a closure around an error
// channel which can be used for asynchronous error signaling from within the
// map function, and handling from the Map call context.
//
// The map function itself attempts to deserialize the given byte slice as a
// the EventsReplayClient's generic typed event, using the decoder function provided.
// If the events bytes observable contained an error, this value is not emitted
// (skipped) on the destination observable of the map operation.
//
// If deserialisation failed because the event bytes were for a different event
// type, this value is also skipped. If deserialisation failed for some other
// reason, this function panics.
func (rClient *replayClient[T]) newMapEventsBytesToTFn(errCh chan<- error) func(
	context.Context,
	either.Bytes,
) (T, bool) {
	return func(
		_ context.Context,
		eitherEventBz either.Bytes,
	) (_ T, skip bool) {
		eventBz, err := eitherEventBz.ValueOrError()
		if err != nil {
			errCh <- err
			// Don't publish (skip) if eitherEventBz contained an error.
			// eitherEventBz should automatically close itself in this case.
			// (i.e. no more values should be mapped to this transformFn's respective
			// dstObservable).
			return *new(T), true
		}

		// attempt to decode the event bytes using the decoder function provided
		// during the EventsReplayClient's construction.
		event, err := rClient.eventDecoder(eventBz)
		if err != nil {
			if ErrEventsUnmarshalEvent.Is(err) {
				// Don't publish (skip) if the message was not the correct event.
				return *new(T), true
			}

			panic(fmt.Sprintf(
				"unexpected error deserialising event: %v; eventBz: %s",
				err, string(eventBz),
			))
		}
		return event, false
	}
}
