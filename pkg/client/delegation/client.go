package delegation

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
	// NB: Subscribing to custom event types is similar to comet events but
	// requires a different query string. Here we are subscribing to all events
	// with the `pocket.application.EventDelegateeChange` type. All events
	// emit three common fields: `action` (event type), `module`, and `sender`.
	// (see: https://docs.cosmos.network/main/learn/advanced/events#default-events)
	delegateeChangeEventQuery = "tm.event='Tx' AND message.action='pocket.application.EventDelegateeChange'"
)

var _ client.DelegationClient = (*delegationClient)(nil)

type delegationClient struct {
	// endpointURL is the URL of RPC endpoint which eventsClient subscription
	// requests will be sent.
	endpointURL string
	// eventsClient is the events query client which is used to subscribe to
	// application delegation change events. It emits an either value which may
	// contain an error, at most, once and closes immediately after if it does.
	eventsClient client.EventsQueryClient
	// latestDelegationObsvbls is an observable which is notified when app
	// delegation change events are received by the events query client
	// subscription created in goPublishDelegationChanges.
	// This observable (and the one it emits) closes when the events bytes
	// observable returns an error and is updated with a new "active" observable
	// after a new events query subscription is created.
	latestDelegationObsvbls observable.Observable[client.DelegateeChange]
	// latestDelegationPublishCh is a channel used to publish the latest
	// delegation change events to the latestDelegationObsvbls observable.
	latestDelegationPublishCh chan<- client.DelegateeChange
}

// eventBytesToDelegateeChangeMapFn is a convenience type to represent the type
// of a function which maps event subscription message bytes into delegation
// event objects. This is used as a transformFn in a channel.Map() call and is
// the type returned by the newEventsBytesToDelegateeChangeMapFn factory function.
type eventBytesToDelegateeChangeMapFn = func(
	context.Context,
	either.Bytes,
) (client.DelegateeChange, bool)

// NewDelegationClient creates a new block client from the given dependencies
// and cometWebsocketURL.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.DelegationClient, error) {
	// Initialise the delegation client
	dClient := &delegationClient{endpointURL: cometWebsocketURL}
	dClient.latestDelegationObsvbls,
		dClient.latestDelegationPublishCh = channel.NewObservable[client.DelegateeChange]()

	// Inject dependencies
	if err := depinject.Inject(deps, &dClient.eventsClient); err != nil {
		return nil, err
	}

	// Concurrently publish delegatee change events to the observable emitted
	// by latestDelegationObsvbls.
	go dClient.goPublishDelegationChanges(ctx)

	return dClient, nil
}

func (dClient *delegationClient) DelegateeChangesObserver(ctx context.Context) client.DelegateeChangesObserver {
	return dClient.latestDelegationObsvbls.Subscribe(ctx)
}

// Close unsubscribes all observers of the committed blocks sequence observable
// and closes the events query client.
func (dClient *delegationClient) Close() {
	// Closing eventsClient will cascade unsubscribe and close downstream observers.
	dClient.eventsClient.Close()
}

// goPublishDelegationChanges runs the work function returned by retryPublishBlocksFactory,
// re-invoking it according to the arguments to retry.OnError when the events bytes
// observable returns an asynchronous error.
// This function is intended to be called in a goroutine.
func (dClient *delegationClient) goPublishDelegationChanges(ctx context.Context) {
	// React to errors by getting a new events bytes observable, re-mapping it,
	// and send it to latestBlockObsvblsReplayPublishCh such that
	// latestDelegationObsvbls.Last(ctx, 1) will return it.
	publishErr := retry.OnError(
		ctx,
		eventsBytesRetryLimit,
		eventsBytesRetryDelay,
		eventsBytesRetryResetTimeout,
		"goPublishDelegationChanges",
		dClient.retryPublishDelegateeChangesFactory(ctx),
	)

	// If we get here, the retry limit was reached and the retry loop exited.
	// Since this function runs in a goroutine, we can't return the error to the
	// caller. Instead, we panic.
	if publishErr != nil {
		panic(fmt.Errorf(
			"DelegationClient.goPublishDelegationChanges should never reach this spot: %w",
			publishErr,
		))
	}
}

// retryPublishDelegateeChangesFactory returns a function which is intended to
// be passed to retry.OnError. The returned function subscribes to event bytes
// from the events query client, converts them to delegatee change events, and
// publishes them to the latestDelegationObsvbls observable.
func (dClient *delegationClient) retryPublishDelegateeChangesFactory(ctx context.Context) func() chan error {
	return func() chan error {
		errCh := make(chan error, 1)
		eventsBzObsvbl, err := dClient.eventsClient.EventsBytes(ctx, delegateeChangeEventQuery)
		if err != nil {
			errCh <- err
			return errCh
		}

		// Subscribe to the events channel and get a read only channel of events
		eventsBzCh := eventsBzObsvbl.Subscribe(ctx).Ch()
		// Continuously read from the events channel until we hit an error
		for {
			eventsBz := <-eventsBzCh
			// Convert the eventsBz into a delegatee change event
			delegateeChange, err := newEventsBytesToDelegateeChange(ctx, eventsBz)
			// If we had an unexpected error, the eventsBz was an error then
			// publish it to the error channel and break from the loop to
			// retry via the retry.OnError function calling this factory fn
			if err != nil {
				errCh <- err
				break
			}
			// Skip if we had a nil delegatee change
			if delegateeChange == nil {
				continue
			}
			// Publish the delegatee change event to the observable's publish channel
			dClient.latestDelegationPublishCh <- delegateeChange
		}

		return errCh
	}
}

// newEventsBytesToDelegateeChange converts the given either value received
// from the EventsQueryClient into a DelegateeChange interface ready to be
// published to the latestDelegationObsvbls observable.
func newEventsBytesToDelegateeChange(
	_ context.Context,
	eitherEventBz either.Bytes,
) (client.DelegateeChange, error) {
	eventBz, err := eitherEventBz.ValueOrError()
	if err != nil {
		return nil, err
	}
	delegateeChange, err := newDelegateeChangeEvent(eventBz)
	if err != nil {
		if ErrUnmarshalDelegateeChangeEvent.Is(err) {
			return nil, nil
		}
		// Only return errors when deserialisation fails
		return nil, fmt.Errorf(
			"unexpected error deserialising delegatee change event: %w; eventBz: %s",
			err, string(eventBz),
		)
	}
	return delegateeChange, nil
}
