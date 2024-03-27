package block

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"

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
	cometClient cometBlockClient,
	deps depinject.Config,
) (client.BlockClient, error) {
	client, err := events.NewEventsReplayClient[client.Block](
		ctx,
		deps,
		committedBlocksQuery,
		newCometBlockEvent,
		defaultBlocksReplayLimit,
	)
	if err != nil {
		return nil, err
	}

	bClient := &blockClient{
		eventsReplayClient: client,
		latestBlockMu:      &sync.Mutex{},
	}

	go bClient.getLatestBlocks(ctx)

	if err := bClient.getInitialBlock(ctx, cometClient); err != nil {
		return nil, err
	}

	return bClient, nil
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
	// latestBlockMu is a mutex that protects the latestBlock field from concurrent
	// writes from getLatestBlocks and getInitialBlock.
	latestBlockMu *sync.Mutex
	// latestBlock is the last block observed by the blockClient.
	// It is initialized by the getInitialBlock function and then updated by the
	// updateLatestBlock goroutine that listens to the eventsReplayClient.
	latestBlock client.Block
}

// CommittedBlocksSequence returns a replay observable of new block events.
func (b *blockClient) CommittedBlocksSequence(ctx context.Context) client.BlockReplayObservable {
	return b.eventsReplayClient.EventsSequence(ctx)
}

// LastNBlocks returns the last n blocks observed by the BlockClient.
func (b *blockClient) LastNBlocks(ctx context.Context, n int) []client.Block {
	return b.eventsReplayClient.LastNEvents(ctx, n)
}

// LastBlock returns the last blocks observed by the BlockClient.
func (b *blockClient) LastBlock() client.Block {
	b.latestBlockMu.Lock()
	defer b.latestBlockMu.Unlock()
	return b.latestBlock
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all downstream connections.
func (b *blockClient) Close() {
	b.eventsReplayClient.Close()
}

// getLatestBlocks listens to the EventsReplayClient's EventsSequence and updates
// the latestBlock field with the latest block observed.
func (b *blockClient) getLatestBlocks(ctx context.Context) {
	latestBlockCh := b.eventsReplayClient.EventsSequence(ctx).Subscribe(ctx).Ch()
	for block := range latestBlockCh {
		b.latestBlockMu.Lock()
		b.latestBlock = block
		b.latestBlockMu.Unlock()
	}
}

// getInitialBlock fetches the initial block from the chain and sets it as the
// latest block observed by the blockClient.
// This is necessary to ensure that the blockClient has the latest block when
// it is first created.
// If b.latestBlock is already set by the getLatestBlocks goroutine, then this
// function skips setting the initial block.
func (b *blockClient) getInitialBlock(ctx context.Context, client cometBlockClient) error {
	queryBlockResult, err := client.Block(ctx, nil)
	if err != nil {
		return err
	}

	initialBlock := &cometBlockEvent{}
	initialBlock.Data.Value.Block = queryBlockResult.Block
	initialBlock.Data.Value.BlockID = queryBlockResult.BlockID

	b.latestBlockMu.Lock()
	defer b.latestBlockMu.Unlock()

	if b.latestBlock == nil {
		b.latestBlock = initialBlock
	}

	return nil
}

// cometBlockClient is an interface that defines the Block method for the comet
// client. This is used to mock the comet client in the block package.

type cometBlockClient interface {
	Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error)
}
