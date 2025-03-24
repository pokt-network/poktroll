package block

import (
	"context"

	"cosmossdk.io/depinject"

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
	// TODO_TECHDEBT: add a `blocksReplayLimit` field to the blockReplayClient
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
// - client.EventsQueryClient
// - client.BlockQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	opts ...client.BlockClientOption,
) (_ client.BlockClient, err error) {
	ctx, cancel := context.WithCancel(ctx)

	// latestBlockPublishCh is the channel that notifies the latestBlockReplayObs of a
	// new block, whether it comes from a direct query or an event subscription query.
	latestBlockReplayObs, latestBlockPublishCh := channel.NewReplayObservable[client.Block](ctx, 10)
	bClient := &blockReplayClient{
		latestBlockReplayObs: latestBlockReplayObs,
		close:                cancel,
	}

	for _, opt := range opts {
		opt(bClient)
	}

	bClient.eventsReplayClient, err = events.NewEventsReplayClient[client.Block](
		ctx,
		deps,
		committedBlocksQuery,
		UnmarshalNewBlock,
		defaultBlocksReplayLimit,
		events.WithConnRetryLimit[client.Block](bClient.connRetryLimit),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := depinject.Inject(deps, &bClient.onStartQueryClient); err != nil {
		return nil, err
	}

	bClient.asyncForwardBlockEvent(ctx, latestBlockPublishCh)

	if err := bClient.getInitialBlock(ctx, latestBlockPublishCh); err != nil {
		return nil, err
	}

	return bClient, nil
}

// blockReplayClient is BlockClient implementation that combines a CometRPC client
// to get the initial block at start up and an EventsReplayClient that subscribes
// to new committed block events.
// It uses a ReplayObservable to retain and replay past observed blocks.
type blockReplayClient struct {
	// onStartQueryClient is the RPC client that is used to query for the initial block
	// upon blockReplayClient construction. The result of this query is only used if it
	// returns before the eventsReplayClient receives its first event.
	onStartQueryClient client.BlockQueryClient

	// eventsReplayClient is the underlying EventsReplayClient that is used to
	// subscribe to new committed block events. It uses both the Block type
	// and the BlockReplayObservable type as its generic types.
	// These enable the EventsReplayClient to correctly map the raw event bytes
	// to Block objects and to correctly return a BlockReplayObservable
	eventsReplayClient client.EventsReplayClient[client.Block]

	// latestBlockReplayObs is a replay observable that combines blocks observed by
	// the block query client & the events replay client. It is the "canonical"
	// source of block notifications for the blockReplayClient.
	latestBlockReplayObs observable.ReplayObservable[client.Block]

	// close is a function that cancels the context of the blockReplayClient.
	close context.CancelFunc

	// connRetryLimit is the number of times the underlying replay client
	// should retry in the event that it encounters an error or its connection is interrupted.
	// If connRetryLimit is < 0, it will retry indefinitely.
	connRetryLimit int
}

// CommittedBlocksSequence returns a replay observable of new block events.
func (b *blockReplayClient) CommittedBlocksSequence(ctx context.Context) client.BlockReplayObservable {
	return b.latestBlockReplayObs
}

// LastBlock returns the last blocks observed by the blockReplayClient.
func (b *blockReplayClient) LastBlock(ctx context.Context) (block client.Block) {
	// ReplayObservable#Last() is guaranteed to return at least one element since
	// it fetches the latest block using the onStartQueryClient if no blocks have
	// been received yet from the eventsReplayClient.
	return b.latestBlockReplayObs.Last(ctx, 1)[0]
}

// Close closes the underlying websocket connection for the EventsQueryClient
// and closes all downstream connections.
func (b *blockReplayClient) Close() {
	b.eventsReplayClient.Close()
	b.close()
}

// asyncForwardBlockEvent asynchronously observes block event notifications from the
// EventsReplayClient's EventsSequence observable & publishes each to latestBlockPublishCh.
func (b *blockReplayClient) asyncForwardBlockEvent(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) {
	channel.ForEach(ctx, b.eventsReplayClient.EventsSequence(ctx),
		func(ctx context.Context, block client.Block) {
			latestBlockPublishCh <- block
		},
	)
}

// getInitialBlock fetches the latest committed onchain block at the time the
// client starts up, while concurrently waiting for the next block event,
// publishing whichever occurs first to latestBlockPublishCh.
// This is necessary to ensure that the most recent block is available to the
// blockReplayClient when it is first created.
func (b *blockReplayClient) getInitialBlock(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Query the latest block asynchronously.
	blockQueryResultCh := make(chan client.Block)
	queryErrCh := b.queryLatestBlock(ctx, blockQueryResultCh)

	// Wait for either the latest block query response, error, or the first block
	// event to arrive & use whichever occurs first or return an error.
	var initialBlock client.Block
	select {
	case initialBlock = <-blockQueryResultCh:
	case <-b.latestBlockReplayObs.Subscribe(ctx).Ch():
		return nil
	case err := <-queryErrCh:
		return err
	}

	// At this point blockQueryResultCh was the first to receive the first block.
	// Publish the initialBlock to the latestBlockPublishCh.
	latestBlockPublishCh <- initialBlock
	return nil
}

// queryLatestBlock uses comet RPC block client to asynchronously query for
// the latest block. It returns an error channel which may be sent a block query error.
// It is *NOT* intended to be called in a goroutine.
func (b *blockReplayClient) queryLatestBlock(
	ctx context.Context,
	blockQueryResultCh chan<- client.Block,
) <-chan error {
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
