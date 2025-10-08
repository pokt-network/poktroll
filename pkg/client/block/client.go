package block

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cosmossdk.io/depinject"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/hashicorp/go-version"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
)

const (
	// cometNewBlockHeaderQuery is the subscription query for block events.
	// - Uses 'NewBlockHeader' events instead of 'NewBlock' for efficiency
	// - 'NewBlock' has complete data but higher bandwidth requirements
	// - Only header information is needed for most block tracking operations
	// - See: https://docs.cosmos.network/v0.47/learn/advanced/events#default-events
	cometNewBlockHeaderQuery = "tm.event='NewBlockHeader'"

	// defaultBlocksReplayLimit is the number of blocks that the replay
	// observable returned by LastNBlocks() will be able to replay.
	// TODO_TECHDEBT: add a `blocksReplayLimit` field to the blockReplayClient
	// struct that defaults to this but can be overridden via an option.
	defaultBlocksReplayLimit = 100

	// blockUpdateStallThreshold is the duration after which a warning is logged
	// and an alert is raised if no new block has been received.
	// TODO_TECHDEBT: Make this value be fetched from the full node.
	blockUpdateStallThreshold = 60 * time.Second
)

// NewBlockClient creates a new block client from the given dependencies.
//
// It uses a pre-defined cometNewBlockHeaderQuery to subscribe to newly
// committed block events which are mapped to Block objects.
//
// This lightly wraps the EventsReplayClient[Block] generic to correctly mock
// the interface.
//
// Required dependencies:
// - client.BlockQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
) (_ client.BlockClient, err error) {
	ctx, cancel := context.WithCancel(ctx)

	// latestBlockPublishCh is the channel that notifies the latestBlockReplayObs of a
	// new block, whether it comes from a direct query or an event subscription query.
	latestBlockReplayObs, latestBlockPublishCh := channel.NewReplayObservable[client.Block](ctx, 10)
	blockClient := &blockReplayClient{
		latestBlockReplayObs: latestBlockReplayObs,
		blockUpdateNotifyCh:  make(chan struct{}, 1),
		close:                cancel,
	}

	blockClient.eventsReplayClient, err = events.NewEventsReplayClient(
		ctx,
		deps,
		cometNewBlockHeaderQuery,
		UnmarshalNewBlock,
		defaultBlocksReplayLimit,
	)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := depinject.Inject(
		deps,
		&blockClient.onStartQueryClient,
		&blockClient.cometClient,
		&blockClient.logger,
	); err != nil {
		return nil, err
	}

	// Start asynchronously forwarding block events to the latestBlockPublishCh
	blockClient.asyncForwardBlockEvent(ctx, latestBlockPublishCh)

	// Get access to the latest block one time and publish it to the latestBlockPublishCh
	// TODO_CONSIDERATION: Should we make this initializeInitialBlock (sync) instead of getInitialBlockAsync (async)?
	if err := blockClient.getInitialBlockAsync(ctx, latestBlockPublishCh); err != nil {
		return nil, err
	}

	// Initialize the chain version one time to ensure that its never nil
	if err := blockClient.initializeChainVersion(ctx); err != nil {
		return nil, err
	}

	// Start monitoring for stalled block updates in the background
	go blockClient.monitorBlockUpdateStalls(ctx)

	return blockClient, nil
}

// blockReplayClient is BlockClient implementation that combines a CometRPC client
// to get the initial block at start up and an EventsReplayClient that subscribes
// to new committed block events.
// It uses a ReplayObservable to retain and replay past observed blocks.
type blockReplayClient struct {
	logger polylog.Logger

	// onStartQueryClient is the RPC client that is used to query for the initial block
	// upon blockReplayClient construction. The result of this query is only used if it
	// returns before the eventsReplayClient receives its first event.
	onStartQueryClient client.BlockQueryClient

	// cometClient is the CometBFT client used to get ABCI info for chain version.
	cometClient cometclient.Client

	// chainVersion is the version of the chain that the block client is connected to.
	// It is protected by chainVersionMu for concurrent access safety.
	chainVersion   *version.Version
	chainVersionMu sync.RWMutex

	// eventsReplayClient is the underlying EventsReplayClient that is used to
	// subscribe to new committed block events. It uses both the Block type
	// and the BlockReplayObservable type as its generic types.
	// These enable the EventsReplayClient to correctly map the raw event bytes
	// to Block objects and to correctly return a BlockReplayObservable
	eventsReplayClient client.EventsReplayClient[client.Block]

	// chainVersionQueryCancel cancels any ongoing ABCI info request for chain version updates.
	// This ensures that when a new block arrives, we cancel the previous request and start fresh.
	chainVersionQueryCancel context.CancelFunc
	chainVersionCancelMu    sync.Mutex

	// latestBlockReplayObs is a replay observable that combines blocks observed by
	// the block query client & the events replay client. It is the "canonical"
	// source of block notifications for the blockReplayClient.
	latestBlockReplayObs observable.ReplayObservable[client.Block]

	// lastBlockUpdateTimeMillis stores the Unix timestamp (in milliseconds) of when
	// the last block was received. Used atomically for detecting stalled block updates.
	lastBlockUpdateTimeMillis atomic.Int64

	// blockUpdateNotifyCh is used to notify the monitoring goroutine when a new
	// block arrives, allowing it to reset its timer.
	blockUpdateNotifyCh chan struct{}

	// close is a function that cancels the context of the blockReplayClient.
	close context.CancelFunc
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
	// Cancel any ongoing requests to retrieve the chain version
	b.chainVersionCancelMu.Lock()
	if b.chainVersionQueryCancel != nil {
		b.chainVersionQueryCancel()
		b.chainVersionQueryCancel = nil
	}
	b.chainVersionCancelMu.Unlock()

	b.close()
}

// GetChainVersion returns the current chain version.
func (b *blockReplayClient) GetChainVersion() *version.Version {
	b.chainVersionMu.RLock()
	defer b.chainVersionMu.RUnlock()
	return b.chainVersion
}

// initializeChainVersion synchronously initializes the chain version.
// Returns an error if initialization fails.
func (b *blockReplayClient) initializeChainVersion(ctx context.Context) error {
	abciInfo, err := b.cometClient.ABCIInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ABCI info: %w", err)
	}

	chainVersion, err := version.NewVersion(abciInfo.Response.Version)
	if err != nil {
		return fmt.Errorf("failed to parse chain version: %w", err)
	}

	b.chainVersionMu.Lock()
	b.chainVersion = chainVersion
	b.chainVersionMu.Unlock()

	return nil
}

// updateChainVersionAsync updates the chain version in the background.
// Cancels any previous ongoing query and handles errors gracefully.
func (b *blockReplayClient) updateChainVersionAsync(ctx context.Context) {
	// Cancel any ongoing chain version query and start a new one
	b.chainVersionCancelMu.Lock()
	if b.chainVersionQueryCancel != nil {
		b.chainVersionQueryCancel()
	}

	queryCtx, cancel := context.WithCancel(ctx)
	b.chainVersionQueryCancel = cancel
	b.chainVersionCancelMu.Unlock()

	go func() {
		defer cancel()

		abciInfo, err := b.cometClient.ABCIInfo(queryCtx)
		if err != nil {
			b.logger.Debug().Err(err).Msg("failed to get ABCI info for chain version update")
			return
		}

		chainVersion, err := version.NewVersion(abciInfo.Response.Version)
		if err != nil {
			b.logger.Debug().Err(err).Msg("failed to parse chain version")
			return
		}

		b.chainVersionMu.Lock()
		b.chainVersion = chainVersion
		b.chainVersionMu.Unlock()
	}()
}

// asyncForwardBlockEvent does the following:
// - Asynchronously observes block event notifications from the EventsReplayClient's EventsSequence observable
// - Publishes each block to latestBlockPublishCh
// - Updates the chain version on each block
// - Resets the block update timer and updates monitoring metrics
func (b *blockReplayClient) asyncForwardBlockEvent(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) {
	channel.ForEach(ctx, b.eventsReplayClient.EventsSequence(ctx),
		func(ctx context.Context, block client.Block) {
			latestBlockPublishCh <- block
			b.updateChainVersionAsync(ctx)

			// Update block monitoring: reset timer and update metrics
			height := block.Height()
			b.lastBlockUpdateTimeMillis.Store(time.Now().UnixMilli())

			// Update Prometheus metric with current block height
			relayer.CaptureBlockHeight(height)

			// Notify the monitoring goroutine to reset its timer (non-blocking)
			select {
			case b.blockUpdateNotifyCh <- struct{}{}:
			default:
			}
		},
	)
}

// getInitialBlockAsync:
// - Fetches the latest committed onchain block at client startup.
// - Concurrently waits for the next block event.
// - Publishes whichever occurs first to latestBlockPublishCh.
// - Ensures the most recent block is available to blockReplayClient when first created.
func (b *blockReplayClient) getInitialBlockAsync(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Query the latest block asynchronously.
	blockQueryResultCh := make(chan client.Block)
	queryErrCh := b.queryLatestBlockAsync(ctx, blockQueryResultCh)

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

	// Initialize block update monitoring timer
	b.lastBlockUpdateTimeMillis.Store(time.Now().UnixMilli())

	// At this point blockQueryResultCh was the first to receive the first block.
	// Publish the initialBlock to the latestBlockPublishCh.
	latestBlockPublishCh <- initialBlock
	return nil
}

// queryLatestBlockAsync:
// - Uses comet RPC block client to asynchronously query for the latest block.
// - Returns an error channel which may be sent a block query error.
// - *NOT* intended to be called in a goroutine.
func (b *blockReplayClient) queryLatestBlockAsync(
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

		blockResult := CometBlockResult(*queryBlockResult)
		blockQueryResultCh <- &blockResult
	}()

	return errCh
}

// monitorBlockUpdateStalls monitors for stalled block updates and raises alerts.
// It uses a timer that resets on each block update. When the timer expires without
// a block update, it logs a warning with the last known block height.
func (b *blockReplayClient) monitorBlockUpdateStalls(ctx context.Context) {
	timer := time.NewTimer(blockUpdateStallThreshold)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.blockUpdateNotifyCh:
			// Block received - reset the timer
			if !timer.Stop() {
				// Drain the channel if timer already fired
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(blockUpdateStallThreshold)
		case <-timer.C:
			// Timer expired - stall detected
			lastUpdateMillis := b.lastBlockUpdateTimeMillis.Load()
			lastUpdateTime := time.UnixMilli(lastUpdateMillis)
			timeSinceLastUpdate := time.Since(lastUpdateTime)

			// Get the last known block height
			lastBlock := b.LastBlock(ctx)
			lastHeight := lastBlock.Height()

			b.logger.Warn().
				Int64("last_block_height", lastHeight).
				Msgf("Block update stalled: no new block received for %s", timeSinceLastUpdate)

			// Reset timer to continue monitoring
			timer.Reset(blockUpdateStallThreshold)
		}
	}
}
