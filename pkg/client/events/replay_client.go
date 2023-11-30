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
	// observable returns an error.
	eventsBytesRetryLimit        = 10
	eventsBytesRetryResetTimeout = 10 * time.Second
	// latestObsvblsReplayBufferSize is the replay buffer size of the
	// latestObsvbls replay observable which is used to cache the latest observable.
	// It is updated with a new "active" observable when a new
	// events query subscription is created, for example, after a non-persistent
	// connection error.
	latestObsvblsReplayBufferSize = 1
	// latestReplayBufferSize is the replay buffer size of the latest replay
	// observable which is notified when block commit events are received
	// by the events query client subscription created in goPublishEvents.
	latestReplayBufferSize = 1
)

// Enforece the EventsReplayClient interface is implemented by the replayClient type.
var _ client.EventsReplayClient[
	any,
	observable.ReplayObservable[any],
] = (*replayClient[any, observable.ReplayObservable[any]])(nil)

// replayClient implements the EventsReplayClient interface for a generic type T,
// and replay observable for type T.
type replayClient[T any, U observable.ReplayObservable[T]] struct {
	// endpointURL is the URL of RPC endpoint which eventsClient subscription
	// requests will be sent.
	endpointURL string
	// queryString is the query string used to subscribe to events of the
	// desired type.
	// See: https://docs.cosmos.network/main/learn/advanced/events#subscribing-to-events
	// and: https://docs.cosmos.network/main/learn/advanced/events#default-events
	queryString string
	// eventsClient is the events query client which is used to subscribe to
	// newly committed block events. It emits an either value which may contain
	// an error, at most, once and closes immediately after if it does.
	eventsClient client.EventsQueryClient
	// eventBytesToTypeDecoder is a function which decodes event subscription
	// message bytes into the type defined by the EventsReplayClient's generic type
	// parameter.
	eventBytesToTypeDecoder func([]byte) (T, error)
	// latestObsvbls is a replay observable with replay buffer size 1,
	// which holds the "active latest observable" which is notified when
	// new events are received by the events query client subscription
	// created in goPublishEvents. This observable (and the one it emits) closes
	// when the events bytes observable returns an error and is updated with a
	// new "active" observable after a new events query subscription is created.
	latestObsvbls observable.ReplayObservable[U]
	// latestObsvblsReplayPublishCh is the publish channel for latestObsvbls.
	// It's used to set latestObsvbls initially and subsequently update it, for
	// example, when the connection is re-established after erroring.
	latestObsvblsReplayPublishCh chan<- U
}

// NewEventsReplayClient creates a new EventsReplayClient from the given
// dependencies, cometWebsocketURL and subscription query string. It requires a
// decoder function to be provided which decodes event subscription message
// bytes into the type defined by the EventsReplayClient's generic type parameter.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewEventsReplayClient[T any, U observable.ReplayObservable[T]](
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
	queryString string,
	eventBytesToTypeDecoder func([]byte) (T, error),
) (client.EventsReplayClient[T, U], error) {
	// Initialise the mapped client
	rClient := &replayClient[T, U]{
		endpointURL:             cometWebsocketURL,
		queryString:             queryString,
		eventBytesToTypeDecoder: eventBytesToTypeDecoder,
	}
	latestObsvbls,
		latestObsvblsReplayPublishCh := channel.NewReplayObservable[U](
		ctx,
		latestReplayBufferSize,
	)
	rClient.latestObsvbls = observable.ReplayObservable[U](latestObsvbls)
	rClient.latestObsvblsReplayPublishCh = latestObsvblsReplayPublishCh

	// Inject dependencies
	if err := depinject.Inject(deps, &rClient.eventsClient); err != nil {
		return nil, err
	}

	// Concurrently publish blocks to the observable emitted by latestObsvbls.
	go rClient.goPublishEvents(ctx)

	return rClient, nil
}

// EventsSequence returns a ReplayObservable, with a replay buffer size of 1,
// which is notified when new events are received by the events query subscription.
func (rClient *replayClient[T, R]) EventsSequence(ctx context.Context) R {
	// Get the latest events observable from the replay observable. We only ever
	// want the last 1 as any prior latest events observable values are closed.
	// Directly accessing the zeroth index here is safe because the call to Last
	// is guaranteed to return a slice with at least 1 element.
	replayObs := observable.ReplayObservable[R](rClient.latestObsvbls)
	return replayObs.Last(ctx, 1)[0]
}

// LastNEvents returns the latest typed event that's been received by the
// corresponding events query subscription.
// It blocks until at least one event has been received.
func (rClient *replayClient[T, R]) LastNEvents(ctx context.Context, n int) []T {
	return rClient.EventsSequence(ctx).Last(ctx, n)
}

// Close unsubscribes all observers of the committed blocks sequence observable
// and closes the events query client.
func (rClient *replayClient[T, R]) Close() {
	// Closing eventsClient will cascade unsubscribe and close downstream observers.
	rClient.eventsClient.Close()
}

// goPublishEvents runs the work function returned by retryPublishEventsFactory,
// re-invoking it according to the arguments to retry.OnError when the events bytes
// observable returns an asynchronous error.
// This function is intended to be called in a goroutine.
func (rClient *replayClient[T, R]) goPublishEvents(ctx context.Context) {
	// React to errors by getting a new events bytes observable, re-mapping it,
	// and send it to latestObsvblsReplayPublishCh such that
	// latestObsvbls.Last(ctx, 1) will return it.
	publishErr := retry.OnError(
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
	if publishErr != nil {
		panic(fmt.Errorf("EventsReplayClient[%T].goPublishEvents should never reach this spot: %w", *new(T), publishErr))
	}
}

// retryPublishEventsFactory returns a function which is intended to be passed
// to retry.OnError. The returned function pipes event bytes from the events
// query client, maps them to block events, and publishes them to the
// latestObsvbls replay observable.
func (rClient *replayClient[T, R]) retryPublishEventsFactory(ctx context.Context) func() chan error {
	return func() chan error {
		errCh := make(chan error, 1)
		eventsBzObsvbl, err := rClient.eventsClient.EventsBytes(ctx, rClient.queryString)
		if err != nil {
			errCh <- err
			return errCh
		}

		// NB: must cast back to generic observable type to use with Map.
		// client.BlocksObservable cannot be an alias due to gomock's lack of
		// support for generic types.
		eventsBz := observable.Observable[either.Either[[]byte]](eventsBzObsvbl)
		typedObsrvbl := channel.MapReplay(
			ctx,
			latestReplayBufferSize,
			eventsBz,
			rClient.newEventsBytesToTypeMapFn(errCh),
		)

		// Initially set latestObsvbls and update if after retrying on error.
		rClient.latestObsvblsReplayPublishCh <- typedObsrvbl.(R)

		return errCh
	}
}

// newEventsBytesToTypeMapFn is a factory for a function which is intended
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
func (rClient *replayClient[T, R]) newEventsBytesToTypeMapFn(errCh chan<- error) func(
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
		event, err := rClient.eventBytesToTypeDecoder(eventBz)
		if err != nil {
			if ErrEventsUnmarshalEvent.Is(err) {
				// Don't publish (skip) if the message was not the correct event.
				return *new(T), true
			}

			panic(fmt.Sprintf(
				"unexpected error deserialising event: %s; eventBz: %s",
				err, string(eventBz),
			))
		}
		return event, false
	}
}
