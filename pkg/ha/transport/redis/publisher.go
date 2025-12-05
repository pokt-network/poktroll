package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ transport.MinedRelayPublisher = (*StreamsPublisher)(nil)

// StreamsPublisher implements MinedRelayPublisher using Redis Streams.
// It publishes mined relays to supplier-specific streams for consumption by the Miner service.
type StreamsPublisher struct {
	logger polylog.Logger
	client redis.UniversalClient
	config transport.PublisherConfig

	// mu protects closed state
	mu     sync.RWMutex
	closed bool
}

// NewStreamsPublisher creates a new Redis Streams publisher.
func NewStreamsPublisher(
	logger polylog.Logger,
	client redis.UniversalClient,
	config transport.PublisherConfig,
) *StreamsPublisher {
	return &StreamsPublisher{
		logger: logging.ForComponent(logger, logging.ComponentRedisPublisher),
		client: client,
		config: config,
	}
}

// Publish sends a mined relay message to the Redis Stream for the supplier.
func (p *StreamsPublisher) Publish(ctx context.Context, msg *transport.MinedRelayMessage) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	// Set published timestamp if not already set
	if msg.PublishedAtUnixNano == 0 {
		msg.SetPublishedAt()
	}

	streamName := transport.StreamName(p.config.StreamPrefix, msg.SupplierOperatorAddress)

	// Serialize message to JSON for Redis Stream
	// Using JSON for human readability and debugging; can switch to protobuf for efficiency
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Build XADD arguments
	args := &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data": data,
		},
	}

	// Apply max length trimming if configured
	if p.config.MaxLen > 0 {
		if p.config.ApproxMaxLen {
			args.Approx = true
		}
		args.MaxLen = p.config.MaxLen
	}

	// Publish to stream
	messageID, err := p.client.XAdd(ctx, args).Result()
	if err != nil {
		publishErrorsTotal.WithLabelValues(msg.SupplierOperatorAddress, msg.ServiceId).Inc()
		return fmt.Errorf("failed to publish to stream %s: %w", streamName, err)
	}

	// Update metrics
	publishedTotal.WithLabelValues(msg.SupplierOperatorAddress, msg.ServiceId).Inc()

	p.logger.Debug().
		Str(logging.FieldStreamID, streamName).
		Str(logging.FieldMessageID, messageID).
		Str(logging.FieldSessionID, msg.SessionId).
		Str(logging.FieldSupplier, msg.SupplierOperatorAddress).
		Msg("published mined relay to stream")

	return nil
}

// PublishBatch sends multiple mined relay messages in a single pipeline operation.
func (p *StreamsPublisher) PublishBatch(ctx context.Context, msgs []*transport.MinedRelayMessage) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("publisher is closed")
	}
	p.mu.RUnlock()

	if len(msgs) == 0 {
		return nil
	}

	// Group messages by supplier for efficient pipelining
	bySupplier := make(map[string][]*transport.MinedRelayMessage)
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		bySupplier[msg.SupplierOperatorAddress] = append(bySupplier[msg.SupplierOperatorAddress], msg)
	}

	// Use pipeline for batch efficiency
	pipe := p.client.Pipeline()
	var cmds []*redis.StringCmd

	for supplier, supplierMsgs := range bySupplier {
		streamName := transport.StreamName(p.config.StreamPrefix, supplier)

		for _, msg := range supplierMsgs {
			// Set published timestamp
			if msg.PublishedAtUnixNano == 0 {
				msg.SetPublishedAt()
			}

			data, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("failed to serialize message: %w", err)
			}

			args := &redis.XAddArgs{
				Stream: streamName,
				Values: map[string]interface{}{
					"data": data,
				},
			}

			if p.config.MaxLen > 0 {
				if p.config.ApproxMaxLen {
					args.Approx = true
				}
				args.MaxLen = p.config.MaxLen
			}

			cmds = append(cmds, pipe.XAdd(ctx, args))
		}
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		// Count errors per supplier
		for supplier, supplierMsgs := range bySupplier {
			for _, msg := range supplierMsgs {
				publishErrorsTotal.WithLabelValues(supplier, msg.ServiceId).Inc()
			}
		}
		return fmt.Errorf("failed to execute batch publish: %w", err)
	}

	// Verify all commands succeeded and update metrics
	for i, cmd := range cmds {
		if cmd.Err() != nil {
			return fmt.Errorf("batch publish command %d failed: %w", i, cmd.Err())
		}
	}

	// Update success metrics
	for supplier, supplierMsgs := range bySupplier {
		for _, msg := range supplierMsgs {
			publishedTotal.WithLabelValues(supplier, msg.ServiceId).Inc()
		}
	}

	p.logger.Debug().
		Int("batch_size", len(msgs)).
		Int("suppliers", len(bySupplier)).
		Msg("published batch of mined relays")

	return nil
}

// Close gracefully shuts down the publisher.
func (p *StreamsPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.logger.Info().Msg("Redis Streams publisher closed")
	return nil
}
