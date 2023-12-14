package delegation

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

const (
	// delegationEventQuery is the query used by the EventsQueryClient to subscribe
	// to new delegation events from the the application module on chain.
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	// And: https://docs.cosmos.network/v0.47/learn/advanced/events#default-events
	delegationEventQuery = "message.action='pocket.application.EventRedelegation'"
	// TODO_TECHDEBT/TODO_FUTURE: add a `redelegationsReplayLimit` field to the
	// delegation client struct that defaults to this but can be overridden via
	// an option in future work.
	// defaultRedelegationsReplayLimit is the number of redelegations that the
	// replay observable returned by LastNRedelegations() will be able to replay.
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
) (client.DelegationClient, error) {
	client, err := events.NewEventsReplayClient[
		client.Redelegation,
		client.EventsObservable[client.Redelegation],
	](
		ctx,
		deps,
		delegationEventQuery,
		newRedelegationEventFactoryFn(),
		defaultRedelegationsReplayLimit,
	)
	if err != nil {
		return nil, err
	}
	return &delegationClient{eventsReplayClient: client}, nil
}

// delegationClient is a wrapper around an EventsReplayClient that implements
// the DelegationClient interface for use with cosmos-sdk networks.
type delegationClient struct {
	// eventsReplayClient is the underlying EventsReplayClient that is used to
	// subscribe to new delegation events. It uses both the Redelegation type
	// and the RedelegationReplayObservable type as its generic types.
	// These enable the EventsReplayClient to correctly map the raw event bytes
	// to Redelegation objects and to correctly return a RedelegationReplayObservable
	eventsReplayClient client.EventsReplayClient[client.Redelegation, client.EventsObservable[client.Redelegation]]
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
