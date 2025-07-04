package block

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/hashicorp/go-version"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
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

	// SigningPayloadHashVersion is the version of the chain that introduced the
	// payload hash in RelayResponse.
	// This is used to determine whether to compute and include the payload hash in
	// the RelayResponse based on the chain version.
	signingPayloadHashVersion = "v0.1.25"
)

// TODO(v0.1.26): Remove this after the entire network is on v0.1.26.
//
// ChainVersionAddPayloadHashInRelayResponse is the version of the chain that:
// - Introduced the payload hash in RelayResponse.
// - Removed the full payload from RelayResponse.
var ChainVersionAddPayloadHashInRelayResponse *version.Version

func init() {
	var err error
	if ChainVersionAddPayloadHashInRelayResponse, err = version.NewVersion("v0.1.25"); err != nil {
		panic("failed to parse chain version add payload hash in relay response: " + err.Error())
	}
}

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

	blockClient.asyncForwardBlockEvent(ctx, latestBlockPublishCh)

	if err := blockClient.getInitialBlock(ctx, latestBlockPublishCh); err != nil {
		return nil, err
	}

	// Initialize the chain version to ensure that its never nil
	if err := blockClient.initializeAndUpdateChainVersion(ctx); err != nil {
		return nil, err
	}

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

// updateChainVersion updates the chain version by querying ABCI info.
// If async is true, it runs in a background goroutine and handles cancellation of previous queries.
// If async is false, it runs synchronously and is suitable for initialization.
func (b *blockReplayClient) updateChainVersion(ctx context.Context, async bool) error {
	var queryCtx context.Context
	var cancel context.CancelFunc

	if async {
		// Cancel any ongoing chain version query and start a new one
		b.chainVersionCancelMu.Lock()
		if b.chainVersionQueryCancel != nil {
			b.chainVersionQueryCancel()
		}

		// Create new context for the chain version query
		queryCtx, cancel = context.WithCancel(ctx)
		b.chainVersionQueryCancel = cancel
		b.chainVersionCancelMu.Unlock()
	} else {
		// For synchronous calls (like initialization), use the provided context directly
		queryCtx = ctx
		cancel = func() {} // No-op cancel function
	}

	updateFunc := func() error {
		if async {
			defer cancel() // Always cleanup the context when done for async calls
		}

		// Update chain version
		abciInfo, err := b.cometClient.ABCIInfo(queryCtx)
		if err != nil {
			if async {
				// Log error but don't stop the process (context may have been cancelled)
				b.logger.Debug().Err(err).Msg("failed to get ABCI info for chain version update")
				return nil // Don't return error for async calls
			}
			return fmt.Errorf("failed to get ABCI info: %w", err)
		}

		chainVersion, err := version.NewVersion(abciInfo.Response.Version)
		if err != nil {
			if async {
				// Log error but don't stop the process
				b.logger.Debug().Err(err).Msg("failed to parse chain version")
				return nil // Don't return error for async calls
			}
			return fmt.Errorf("failed to parse chain version: %w", err)
		}

		// Update chain version with mutex protection
		b.chainVersionMu.Lock()
		b.chainVersion = chainVersion
		b.chainVersionMu.Unlock()

		return nil
	}

	if async {
		go updateFunc()
		return nil
	}

	return updateFunc()
}

// asyncForwardBlockEvent does the following:
// - Asynchronously observes block event notifications from the EventsReplayClient's EventsSequence observable
// - Publishes each block to latestBlockPublishCh
// - Updates the chain version on each block
func (b *blockReplayClient) asyncForwardBlockEvent(
	ctx context.Context,
	latestBlockPublishCh chan<- client.Block,
) {
	channel.ForEach(ctx, b.eventsReplayClient.EventsSequence(ctx),
		func(ctx context.Context, block client.Block) {
			latestBlockPublishCh <- block
			b.updateChainVersion(ctx, true) // async=true
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

		blockResult := CometBlockResult(*queryBlockResult)
		blockQueryResultCh <- &blockResult
	}()

	return errCh
}
