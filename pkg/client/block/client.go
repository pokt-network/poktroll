package block

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

const (
	// committedBlocksQuery is the query used to subscribe to new committed block
	// events used by the EventsQueryClient to subscribe to new block events from
	// the chain.
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#default-events
	committedBlocksQuery = "tm.event='NewBlock'"

	// defaultBlocksReplayLimit is the number of blocks that the replay
	// observable returned by LastNBlocks() will be able to replay.
	// TODO_TECHDEBT/TODO_FUTURE: add a `blocksReplayLimit` field to the blockClient
	// struct that defaults to this but can be overridden via an option.
	defaultBlocksReplayLimit = 100
)

// NewBlockClient creates a new block client from the given dependencies and
// cometWebsocketURL. It uses a pre-defined committedBlocksQuery to subscribe to
// newly committed block events which are mapped to Block objects.
//
// This lightly wraps the EventsReplayClient[Block] generic to correctly mock
// the interface.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
) (client.BlockClient, error) {
	client, err := events.NewEventsReplayClient[client.Block](
		ctx,
		deps,
		committedBlocksQuery,
		newCometBlockEventFactoryFn(),
		defaultBlocksReplayLimit,
	)
	if err != nil {
		return nil, err
	}
	return &blockClient{eventsReplayClient: client}, nil
}

// blockClient is a wrapper around an EventsReplayClient that implements the
// BlockClient interface for use with cosmos-sdk networks.
type blockClient struct {
	// eventsReplayClient is the underlying EventsReplayClient that is used to
	// subscribe to new committed block events. It uses both the Block type
	// and the BlockReplayObservable type as its generic types.
	// These enable the EventsReplayClient to correctly map the raw event bytes
	// to Block objects and to correctly return a BlockReplayObservable
	eventsReplayClient client.EventsReplayClient[client.Block]
}

// CommittedBlocksSequence returns a replay observable of new block events.
func (b *blockClient) CommittedBlocksSequence(ctx context.Context) client.BlockReplayObservable {
	return b.eventsReplayClient.EventsSequence(ctx)
}

// LatestsNBlocks returns the last n blocks observed by the BockClient.
func (b *blockClient) LastNBlocks(ctx context.Context, n int) []client.Block {
	return b.eventsReplayClient.LastNEvents(ctx, n)
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all downstream connections.
func (b *blockClient) Close() {
	b.eventsReplayClient.Close()
}
