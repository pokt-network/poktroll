package block

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	replayclient "github.com/pokt-network/poktroll/pkg/client/events_replay_client"
)

// committedBlocksQuery is the query used to subscribe to new committed block
// events used by the EventsQueryClient to subscribe to new block events from
// the chain.
// See: https://docs.cosmos.network/main/learn/advanced/events#subscribing-to-events
const committedBlocksQuery = "tm.event='NewBlock'"

// NewBlockClient creates a new block client from the given dependencies and
// cometWebsocketURL. It uses the defined committedBlocksQuery to subscribe to
// newly committed block events and maps them to Block objects, using the
// newCometBlockEvent function as the mapping function.
//
// This is an implementation of the EventsReplayClient[Block] generic.
// correctly mock the interface.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (Client, error) {
	client, err := replayclient.NewEventsReplayClient[
		client.Block,
		client.EventsObservable[client.Block],
	](
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

// blockClient is a wrapper around an EventsReplayClient that implements the
// interface for use in network.
type blockClient struct {
	mappedClient client.EventsReplayClient[client.Block, client.EventsObservable[client.Block]]
}

// CommittedBlocksSequence returns a replay observable of observables for Block events
// from the BlockClient.
func (b *blockClient) CommittedBlocksSequence(ctx context.Context) Observable {
	return b.mappedClient.EventsSequence(ctx).(Observable)
}

// LatestsNEvents returns the latest n blocks from the BockClient.
func (b *blockClient) LastNBlocks(ctx context.Context, n int) []client.Block {
	events := b.mappedClient.LastNEvents(ctx, n)
	for _, event := range events {
		// Casting here is safe as this is the generic type of
		// the EventsReplayClient
		event = event.(client.Block)
	}
	return events
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all subsequent connections.
func (b *blockClient) Close() {
	b.mappedClient.Close()
}
