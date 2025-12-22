package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/hashicorp/go-version"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// BlockPollerConfig contains configuration for the block poller.
type BlockPollerConfig struct {
	// RPCEndpoint is the CometBFT RPC endpoint (e.g., "http://localhost:26657")
	RPCEndpoint string

	// PollInterval is how often to poll for new blocks.
	// Default: 1 second
	PollInterval time.Duration

	// UseTLS enables TLS for the RPC connection.
	UseTLS bool
}

// DefaultBlockPollerConfig returns sensible defaults.
func DefaultBlockPollerConfig() BlockPollerConfig {
	return BlockPollerConfig{
		PollInterval: time.Second,
	}
}

// simpleBlock implements client.Block interface.
type simpleBlock struct {
	height int64
	hash   []byte
}

func (b *simpleBlock) Height() int64 { return b.height }
func (b *simpleBlock) Hash() []byte  { return b.hash }

// BlockPoller is a simplified BlockClient that polls for block updates.
// It implements client.BlockClient interface but uses polling instead of websocket subscriptions.
type BlockPoller struct {
	logger      polylog.Logger
	config      BlockPollerConfig
	cometClient *http.HTTP

	// Current block state
	lastBlock   atomic.Pointer[simpleBlock]
	lastBlockMu sync.RWMutex

	// Chain version
	chainVersion   *version.Version
	chainVersionMu sync.RWMutex

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	closed   bool
	mu       sync.Mutex
}

// NewBlockPoller creates a new block poller.
func NewBlockPoller(
	logger polylog.Logger,
	config BlockPollerConfig,
) (*BlockPoller, error) {
	if config.RPCEndpoint == "" {
		return nil, fmt.Errorf("RPC endpoint is required")
	}
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}

	// Create CometBFT HTTP client
	var cometClient *http.HTTP
	var err error

	if config.UseTLS {
		// For TLS, we need to create a custom HTTP client
		httpClient := &tls.Config{MinVersion: tls.VersionTLS12}
		_ = httpClient // TODO: Use custom transport if needed
		cometClient, err = http.New(config.RPCEndpoint, "/websocket")
	} else {
		cometClient, err = http.New(config.RPCEndpoint, "/websocket")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create CometBFT client: %w", err)
	}

	return &BlockPoller{
		logger:      logging.ForComponent(logger, logging.ComponentBlockPoller),
		config:      config,
		cometClient: cometClient,
	}, nil
}

// Start begins polling for new blocks.
func (bp *BlockPoller) Start(ctx context.Context) error {
	bp.mu.Lock()
	if bp.closed {
		bp.mu.Unlock()
		return fmt.Errorf("block poller is closed")
	}
	bp.ctx, bp.cancelFn = context.WithCancel(ctx)
	bp.mu.Unlock()

	// Get initial block
	if err := bp.fetchLatestBlock(ctx); err != nil {
		bp.logger.Warn().Err(err).Msg("failed to fetch initial block, will retry")
	}

	// Initialize chain version
	if err := bp.initializeChainVersion(ctx); err != nil {
		bp.logger.Warn().Err(err).Msg("failed to initialize chain version")
	}

	// Start polling goroutine
	bp.wg.Add(1)
	go bp.pollLoop(bp.ctx)

	bp.logger.Info().
		Str("rpc_endpoint", bp.config.RPCEndpoint).
		Dur("poll_interval", bp.config.PollInterval).
		Msg("block poller started")

	return nil
}

// pollLoop polls for new blocks at the configured interval.
func (bp *BlockPoller) pollLoop(ctx context.Context) {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := bp.fetchLatestBlock(ctx); err != nil {
				bp.logger.Debug().Err(err).Msg("failed to fetch latest block")
			}
		}
	}
}

// fetchLatestBlock fetches and stores the latest block.
func (bp *BlockPoller) fetchLatestBlock(ctx context.Context) error {
	result, err := bp.cometClient.Block(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to query block: %w", err)
	}

	block := &simpleBlock{
		height: result.Block.Height,
		hash:   result.Block.Hash(),
	}

	oldBlock := bp.lastBlock.Load()
	bp.lastBlock.Store(block)

	// Log if height changed
	if oldBlock == nil || block.height > oldBlock.height {
		bp.logger.Debug().
			Int64("height", block.height).
			Msg("new block received")
	}

	return nil
}

// initializeChainVersion fetches and stores the chain version.
func (bp *BlockPoller) initializeChainVersion(ctx context.Context) error {
	abciInfo, err := bp.cometClient.ABCIInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ABCI info: %w", err)
	}

	chainVersion, err := version.NewVersion(abciInfo.Response.Version)
	if err != nil {
		return fmt.Errorf("failed to parse chain version: %w", err)
	}

	bp.chainVersionMu.Lock()
	bp.chainVersion = chainVersion
	bp.chainVersionMu.Unlock()

	bp.logger.Info().
		Str("version", chainVersion.String()).
		Msg("chain version initialized")

	return nil
}

// LastBlock returns the last known block.
func (bp *BlockPoller) LastBlock(ctx context.Context) client.Block {
	block := bp.lastBlock.Load()
	if block == nil {
		// If no block yet, try to fetch one
		_ = bp.fetchLatestBlock(ctx)
		block = bp.lastBlock.Load()
		if block == nil {
			// Return a zero block if still nil
			return &simpleBlock{height: 0, hash: nil}
		}
	}
	return block
}

// CommittedBlocksSequence returns nil since we don't use subscriptions.
// The SessionLifecycleManager uses polling via LastBlock instead.
func (bp *BlockPoller) CommittedBlocksSequence(ctx context.Context) client.BlockReplayObservable {
	// Not implemented for polling-based client
	return nil
}

// GetChainVersion returns the cached chain version.
func (bp *BlockPoller) GetChainVersion() *version.Version {
	bp.chainVersionMu.RLock()
	defer bp.chainVersionMu.RUnlock()
	return bp.chainVersion
}

// GetChainID fetches the chain ID from the node.
func (bp *BlockPoller) GetChainID(ctx context.Context) (string, error) {
	status, err := bp.cometClient.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get node status: %w", err)
	}
	return status.NodeInfo.Network, nil
}

// Close stops the block poller.
func (bp *BlockPoller) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed {
		return
	}
	bp.closed = true

	if bp.cancelFn != nil {
		bp.cancelFn()
	}

	bp.wg.Wait()

	bp.logger.Info().Msg("block poller closed")
}

// Ensure BlockPoller implements client.BlockClient
var _ client.BlockClient = (*BlockPoller)(nil)
