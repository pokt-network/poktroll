package delegation

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	mappedclient "github.com/pokt-network/poktroll/pkg/client/mapped_client"
)

// delegationEventQuery is the query used by the EventsQueryClien to subscribe
// to new delegation events from the the application module on chain.
// See: https://docs.cosmos.network/main/learn/advanced/events#subscribing-to-events
// And: https://docs.cosmos.network/main/learn/advanced/events#default-events
const delegationEventQuery = "tm.event='Tx' AND message.action='pocket.application.EventDelegateeChange'"

// NewDelegationClient creates a new delegation client from the given
// dependencies and cometWebsocketURL. It uses the defined delegationEventQuery
// to subscribe to new delegation events and maps them to DelegateeChange
// objects. This is an implementation of the MappedClient[DelegateeChange]
// generic type wrapped in order for gomock to correctly mock the interface.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewDelegationClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.DelegationClient, error) {
	client, err := mappedclient.NewMappedClient[client.DelegateeChange, client.EventsObservable[client.DelegateeChange]](
		ctx,
		deps,
		cometWebsocketURL,
		delegationEventQuery,
		newDelegateeChangeEvent,
	)
	if err != nil {
		return nil, err
	}
	return &delegationClient{mappedClient: client}, nil
}

// delegationClient is a wrapper around a mapped client that implements the same
// interface for use in network. This is due to the lack of support from
// gomock for generic types.
type delegationClient struct {
	mappedClient client.MappedClient[client.DelegateeChange, client.EventsObservable[client.DelegateeChange]]
}

// EventsSequence returns a replay observable of observables for delegation events
// from the DelegationClient.
func (b *delegationClient) EventsSequence(ctx context.Context) client.DelegateeChangeObservable {
	return b.mappedClient.EventsSequence(ctx).(client.DelegateeChangeObservable)
}

// LatestsNEvents returns the latest n delegatee change events from the DelegationClient.
func (b *delegationClient) LastNEvents(ctx context.Context, n int) []client.DelegateeChange {
	events := b.mappedClient.LastNEvents(ctx, n)
	for _, event := range events {
		// Casting here is safe as this is the generic type of the MappedClient
		event = event.(client.DelegateeChange)
	}
	return events
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all subsequent connections.
func (b *delegationClient) Close() {
	b.mappedClient.Close()
}
