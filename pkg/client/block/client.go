package block

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	mappedclient "github.com/pokt-network/poktroll/pkg/client/mapped_client"
)

// committedBlocksQuery is the query used to subscribe to new block evnets.
// See: https://docs.cosmos.network/main/learn/advanced/events#subscribing-to-events
const committedBlocksQuery = "tm.event='NewBlock'"

var _ client.BlockClient = (*blockClient)(nil)

// NewBlockClient creates a new block client from the given dependencies and
// cometWebsocketURL. It uses the defined committedBlocksQuery to subscribe to
// newly committed block events and maps them to Block objects.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.BlockClient, error) {
	client, err := mappedclient.NewMappedClient[client.Block, client.EventsObservable[client.Block]](
		ctx,
		deps,
		cometWebsocketURL,
		committedBlocksQuery,
		newCometBlockEvent,
	)
	if err != nil {
		return nil, err
	}
	return &blockClient{mappedClient: client}, nil
}

// blockClient is a wrapper around a mapped client that implements the same
// interface for use in network. This is due to the lack of support from
// gomock for generic types.
type blockClient struct {
	mappedClient client.MappedClient[client.Block, client.EventsObservable[client.Block]]
}

// EventsSequence returns a replay observable of observables for Block events
// from the BlockClient.
func (b *blockClient) EventsSequence(ctx context.Context) client.BlockObservable {
	return b.mappedClient.EventsSequence(ctx).(client.BlockObservable)
}

// LatestsNEvents returns the latest n blocks from the BockClient.
func (b *blockClient) LastNEvents(ctx context.Context, n int) []client.Block {
	events := b.mappedClient.LastNEvents(ctx, n)
	for _, event := range events {
		// Casting here is safe as this is the generic type of the MappedClient
		event = event.(client.Block)
	}
	return events
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all subsequent connections.
func (b *blockClient) Close() {
	b.mappedClient.Close()
}
