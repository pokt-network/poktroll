package delegation

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

const (
	// delegationEventQuery is the query used by the EventsQueryClient to subscribe
	// to all application module events in order to filter for redelegation events.
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	// And: https://docs.cosmos.network/v0.47/learn/advanced/events#default-events
	// TODO_HACK(#280): Instead of listening to all events and doing a verbose
	// filter, we should subscribe to both MsgDelegateToGateway and MsgUndelegateFromGateway
	// messages directly, and filter those for the EventRedelegation event types.
	// This would save the delegation client from listening to a lot of unnecessary
	// events, that it filters out.
	// NB: This is not currently possible because the observer pattern does not
	// support multiplexing multiple observables into a single observable, that
	// can supply the EventsReplayClient with both the MsgDelegateToGateway and
	// MsgUndelegateFromGateway events.
	delegationEventQuery = "tm.event='Tx' AND message.module='application'"

	// defaultRedelegationsReplayLimit is the number of redelegations that the
	// replay observable returned by LastNRedelegations() will be able to replay.
	// TODO_TECHDEBT: add a `redelegationsReplayLimit` field to the
	// delegation client struct that defaults to this but can be overridden via
	// an option in future work.
	defaultRedelegationsReplayLimit = 100
)

// NewDelegationClient creates a new delegation client from the given
// dependencies and cometWebsocketURL. It uses a pre-defined delegationEventQuery
// to subscribe to newly emitted redelegation events which are mapped to
// Redelegation objects.
//
// This lightly wraps the EventsReplayClient[Redelegation] generic to
// correctly mock the interface.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewDelegationClient(
	ctx context.Context,
	deps depinject.Config,
	opts ...client.DelegationClientOption,
) (_ client.DelegationClient, err error) {
	dClient := &delegationClient{}

	for _, opt := range opts {
		opt(dClient)
	}

	dClient.eventsReplayClient, err = events.NewEventsReplayClient[client.Redelegation](
		ctx,
		deps,
		delegationEventQuery,
		newRedelegationEventFactoryFn(),
		defaultRedelegationsReplayLimit,
		events.WithConnRetryLimit[client.Redelegation](dClient.connRetryLimit),
	)
	if err != nil {
		return nil, err
	}

	return dClient, nil
}

// delegationClient is a wrapper around an EventsReplayClient that implements
// the DelegationClient interface for use with cosmos-sdk networks.
type delegationClient struct {
	// eventsReplayClient is the underlying EventsReplayClient that is used to
	// subscribe to new delegation events. It uses both the Redelegation type
	// and the RedelegationReplayObservable type as its generic types.
	// These enable the EventsReplayClient to correctly map the raw event bytes
	// to Redelegation objects and to correctly return a RedelegationReplayObservable
	eventsReplayClient client.EventsReplayClient[client.Redelegation]

	// connRetryLimit is the number of times the underlying replay client
	// should retry in the event that it encounters an error or its connection is interrupted.
	// If connRetryLimit is < 0, it will retry indefinitely.
	connRetryLimit int
}

// RedelegationsSequence returns a replay observable of Redelgation events
// observed by the DelegationClient.
func (b *delegationClient) RedelegationsSequence(ctx context.Context) client.RedelegationReplayObservable {
	return b.eventsReplayClient.EventsSequence(ctx)
}

// LastNRedelegations returns the latest n redelegation events from the DelegationClient.
func (b *delegationClient) LastNRedelegations(ctx context.Context, n int) []client.Redelegation {
	return b.eventsReplayClient.LastNEvents(ctx, n)
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all downstream connections.
func (b *delegationClient) Close() {
	b.eventsReplayClient.Close()
}
