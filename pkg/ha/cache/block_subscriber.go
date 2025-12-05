package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ BlockHeightSubscriber = (*RedisBlockSubscriber)(nil)

// RedisBlockSubscriber implements BlockHeightSubscriber using Redis Pub/Sub.
// It allows multiple Relayer instances to stay synchronized on the current block height.
type RedisBlockSubscriber struct {
	logger      polylog.Logger
	redisClient redis.UniversalClient
	blockClient client.BlockClient
	config      CacheConfig

	// Subscribers
	subscribers   []chan BlockEvent
	subscribersMu sync.RWMutex

	// Current block height (cached locally)
	currentHeight int64
	heightMu      sync.RWMutex

	// Lifecycle
	mu       sync.RWMutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewRedisBlockSubscriber creates a new BlockHeightSubscriber backed by Redis Pub/Sub.
func NewRedisBlockSubscriber(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	blockClient client.BlockClient,
	config CacheConfig,
) *RedisBlockSubscriber {
	if config.PubSubPrefix == "" {
		config.PubSubPrefix = "ha:events"
	}

	return &RedisBlockSubscriber{
		logger:      logging.ForComponent(logger, logging.ComponentBlockSubscriber),
		redisClient: redisClient,
		blockClient: blockClient,
		config:      config,
	}
}

// Start begins listening for block height updates.
func (s *RedisBlockSubscriber) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("subscriber is closed")
	}

	ctx, s.cancelFn = context.WithCancel(ctx)
	s.mu.Unlock()

	// Initialize current height from block client
	if s.blockClient != nil {
		lastBlock := s.blockClient.LastBlock(ctx)
		s.heightMu.Lock()
		s.currentHeight = lastBlock.Height()
		s.heightMu.Unlock()
		currentBlockHeight.Set(float64(s.currentHeight))
	}

	// Subscribe to Redis pub/sub
	s.wg.Add(1)
	go s.subscribeLoop(ctx)

	s.logger.Info().Int64("initial_height", s.currentHeight).Msg("block subscriber started")
	return nil
}

// subscribeLoop listens for block events from Redis Pub/Sub.
func (s *RedisBlockSubscriber) subscribeLoop(ctx context.Context) {
	defer s.wg.Done()

	channel := s.config.PubSubPrefix + ":block"
	pubsub := s.redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-pubsub.Channel():
			var event BlockEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				s.logger.Warn().Err(err).Str("payload", msg.Payload).Msg("invalid block event")
				continue
			}

			s.handleBlockEvent(event)
		}
	}
}

// handleBlockEvent processes a received block event.
func (s *RedisBlockSubscriber) handleBlockEvent(event BlockEvent) {
	// Update current height
	s.heightMu.Lock()
	if event.Height > s.currentHeight {
		s.currentHeight = event.Height
		currentBlockHeight.Set(float64(s.currentHeight))
	}
	s.heightMu.Unlock()

	blockEventsReceived.Inc()

	// Notify all subscribers
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip (subscriber is slow)
			s.logger.Warn().Int64("height", event.Height).Msg("subscriber channel full, dropping event")
		}
	}
}

// Subscribe returns a channel that receives new block heights.
func (s *RedisBlockSubscriber) Subscribe(ctx context.Context) <-chan BlockEvent {
	ch := make(chan BlockEvent, 10)

	s.subscribersMu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.subscribersMu.Unlock()

	// Clean up when context is cancelled
	go func() {
		<-ctx.Done()
		s.unsubscribe(ch)
	}()

	return ch
}

// unsubscribe removes a subscriber channel.
func (s *RedisBlockSubscriber) unsubscribe(ch chan BlockEvent) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for i, sub := range s.subscribers {
		if sub == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// PublishBlockHeight publishes a new block height to all subscribers.
func (s *RedisBlockSubscriber) PublishBlockHeight(ctx context.Context, event BlockEvent) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("subscriber is closed")
	}
	s.mu.RUnlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	channel := s.config.PubSubPrefix + ":block"

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal block event: %w", err)
	}

	if err := s.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish block event: %w", err)
	}

	blockEventsPublished.Inc()

	s.logger.Debug().Int64("height", event.Height).Msg("published block event")
	return nil
}

// GetCurrentHeight returns the current block height (cached locally).
func (s *RedisBlockSubscriber) GetCurrentHeight() int64 {
	s.heightMu.RLock()
	defer s.heightMu.RUnlock()
	return s.currentHeight
}

// Close gracefully shuts down the subscriber.
func (s *RedisBlockSubscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	if s.cancelFn != nil {
		s.cancelFn()
	}

	// Close all subscriber channels
	s.subscribersMu.Lock()
	for _, ch := range s.subscribers {
		close(ch)
	}
	s.subscribers = nil
	s.subscribersMu.Unlock()

	s.wg.Wait()

	s.logger.Info().Msg("block subscriber closed")
	return nil
}

// BlockHeightWatcher watches the blockchain for new blocks and publishes events.
// This should run on a single instance (leader) to avoid duplicate events.
type BlockHeightWatcher struct {
	logger      polylog.Logger
	blockClient client.BlockClient
	subscriber  *RedisBlockSubscriber

	mu       sync.Mutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewBlockHeightWatcher creates a new watcher that publishes block events.
func NewBlockHeightWatcher(
	logger polylog.Logger,
	blockClient client.BlockClient,
	subscriber *RedisBlockSubscriber,
) *BlockHeightWatcher {
	return &BlockHeightWatcher{
		logger:      logger.With("component", "block_watcher"),
		blockClient: blockClient,
		subscriber:  subscriber,
	}
}

// Start begins watching for new blocks and publishing events.
func (w *BlockHeightWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("watcher is closed")
	}

	ctx, w.cancelFn = context.WithCancel(ctx)
	w.mu.Unlock()

	w.wg.Add(1)
	go w.watchLoop(ctx)

	w.logger.Info().Msg("block height watcher started")
	return nil
}

// watchLoop polls for new blocks and publishes events.
func (w *BlockHeightWatcher) watchLoop(ctx context.Context) {
	defer w.wg.Done()

	lastHeight := w.blockClient.LastBlock(ctx).Height()

	ticker := time.NewTicker(time.Second) // Poll every second
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentBlock := w.blockClient.LastBlock(ctx)
			currentHeight := currentBlock.Height()

			if currentHeight > lastHeight {
				// New block(s) - publish events for each
				for h := lastHeight + 1; h <= currentHeight; h++ {
					event := BlockEvent{
						Height:    h,
						Timestamp: time.Now(),
					}

					if h == currentHeight {
						event.Hash = currentBlock.Hash()
					}

					if err := w.subscriber.PublishBlockHeight(ctx, event); err != nil {
						w.logger.Error().Err(err).Int64("height", h).Msg("failed to publish block event")
					}
				}

				lastHeight = currentHeight
			}
		}
	}
}

// Close gracefully shuts down the watcher.
func (w *BlockHeightWatcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	if w.cancelFn != nil {
		w.cancelFn()
	}

	w.wg.Wait()

	w.logger.Info().Msg("block height watcher closed")
	return nil
}
