package block

import (
	"context"

	"cosmossdk.io/depinject"
	cometclient "github.com/cosmos/cosmos-sdk/client"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
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
	ctx, close := context.WithCancel(ctx)

	eventsReplayClient, err := events.NewEventsReplayClient[client.Block](
		ctx,
		deps,
		committedBlocksQuery,
		newCometBlockEvent,
		defaultBlocksReplayLimit,
	)
	if err != nil {
		close()
		return nil, err
	}

	// latestBlockPublishCh is the channel that notifies the latestBlockReplayObs of a
	// new block, whether it comes from a direct query or an event subscription query.
	latestBlockReplayObs, latestBlockPublishCh := channel.NewReplayObservable[client.Block](ctx, 10)
	blockClient := &blockReplayClient{
		eventsReplayClient:   eventsReplayClient,
		latestBlockReplayObs: latestBlockReplayObs,
		close:                close,
	}

	if err := depinject.Inject(deps, &blockClient.onStartQueryClient); err != nil {
		return nil, err
	}

	blockClient.forEachBlockEvent(ctx, latestBlockPublishCh)

	if err := blockClient.getInitialBlock(ctx, latestBlockPublishCh); err != nil {
		return nil, err
	}

	return blockClient, nil
}

// blockReplayClient is a wrapper around an EventsReplayClient that implements the
// BlockClient interface for use with cosmos-sdk networks.
type blockReplayClient struct {
	// onStartQueryClient is the RPC client that is used to query for the initial block
	// upon blockClient construction. The result of this query is only used if it
	// returns before the eventsReplayClient receives its first event.
	onStartQueryClient cometclient.CometRPC

	// eventsReplayClient is the underlying EventsReplayClient that is used to
	// subscribe to new committed block events. It uses both the Block type
	// and the BlockReplayObservable type as its generic types.
	// These enable the EventsReplayClient to correctly map the raw event bytes
	// to Block objects and to correctly return a BlockReplayObservable
	eventsReplayClient client.EventsReplayClient[client.Block]

	// latestBlockReplayObs is a replay observable that combines blocks observed by
	// the block query client & the events replay client. It is the "canonical"
	// source of block notifications for blockClient.
	latestBlockReplayObs observable.ReplayObservable[client.Block]

	// close is a function that cancels the context of the blockClient.
	close context.CancelFunc
}

// CommittedBlocksSequence returns a replay observable of new block events.
func (b *blockReplayClient) CommittedBlocksSequence(ctx context.Context) client.BlockReplayObservable {
	return b.latestBlockReplayObs
}

// LastBlock returns the last blocks observed by the BlockClient.
func (b *blockReplayClient) LastBlock(ctx context.Context) (block client.Block) {
	// ReplayObservable#Last() is guaranteed to return at least one element.
	return b.latestBlockReplayObs.Last(ctx, 1)[0]
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all downstream connections.
func (b *blockReplayClient) Close() {
	b.eventsReplayClient.Close()
	//close(b.latestBlockPublishCh)
	b.close()
}

// forEachBlockEvent asynchronously observes block event notifications from the
// EventsReplayClient's EventsSequence observable & publishes each to latestBlockPublishCh.
func (b *blockReplayClient) forEachBlockEvent(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) {
	channel.ForEach(ctx, b.eventsReplayClient.EventsSequence(ctx),
		func(ctx context.Context, block client.Block) {
			latestBlockPublishCh <- block
		},
	)
}

// getInitialBlock fetches the latest committed on-chain block at the time the
// client starts up, while concurrently waiting for the next block event,
// publishing whichever occurs first to latestBlockPublishCh.
// This is necessary to ensure that the most recent block is available to the
// blockClient when it is first created.
func (b *blockReplayClient) getInitialBlock(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	blockQueryResultCh := make(chan client.Block)

	// Query the latest block asynchronously.
	queryErrCh := b.queryLatestBlock(ctx, blockQueryResultCh)

	// Wait for either the latest block query response, error, or the first block
	// event to arrive & use whichever occurs first or return an error.
	//
	// NB: #latestBlockReplayObs is a proxy for the events sequence observable
	// because it is guaranteed to be notified on block events in #goForEachBLockEvent().
	var initialBlock client.Block
	select {
	case initialBlock = <-blockQueryResultCh:
	case <-b.latestBlockReplayObs.Subscribe(ctx).Ch():
		return nil
	case err := <-queryErrCh:
		return err
	}

	// Publish the fastest result as the initial block.
	latestBlockPublishCh <- initialBlock
	return nil
}

// queryLatestBlock constructs a comet RPC block client & asynchronously queries for
// the latest block. It returns an error channel which may be sent a block query error.
// It is *NOT* intended to be called in a goroutine.
func (b *blockReplayClient) queryLatestBlock(ctx context.Context, blockQueryResultCh chan<- client.Block) <-chan error {
	errCh := make(chan error)

	go func() {
		queryBlockResult, err := b.onStartQueryClient.Block(ctx, nil)
		if err != nil {
			errCh <- err
			return
		}

		blockResult := cometBlockResult(*queryBlockResult)
		blockQueryResultCh <- &blockResult
	}()

	return errCh
}
