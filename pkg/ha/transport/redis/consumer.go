package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ transport.MinedRelayConsumer = (*StreamsConsumer)(nil)

// StreamsConsumer implements MinedRelayConsumer using Redis Streams with consumer groups.
// It provides exactly-once delivery semantics within the consumer group.
type StreamsConsumer struct {
	logger     polylog.Logger
	client     redis.UniversalClient
	config     transport.ConsumerConfig
	streamName string

	// Message channel
	msgCh chan transport.StreamMessage

	// Lifecycle management
	mu       sync.RWMutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewStreamsConsumer creates a new Redis Streams consumer.
func NewStreamsConsumer(
	logger polylog.Logger,
	client redis.UniversalClient,
	config transport.ConsumerConfig,
) (*StreamsConsumer, error) {
	if config.StreamPrefix == "" {
		return nil, fmt.Errorf("stream prefix is required")
	}
	if config.SupplierOperatorAddress == "" {
		return nil, fmt.Errorf("supplier operator address is required")
	}
	if config.ConsumerGroup == "" {
		return nil, fmt.Errorf("consumer group is required")
	}
	if config.ConsumerName == "" {
		return nil, fmt.Errorf("consumer name is required")
	}

	// Set defaults
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.BlockTimeout <= 0 {
		config.BlockTimeout = 5000 // 5 seconds
	}
	if config.ClaimIdleTimeout <= 0 {
		config.ClaimIdleTimeout = 30000 // 30 seconds
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &StreamsConsumer{
		logger:     logging.ForSupplierComponent(logger, logging.ComponentRedisConsumer, config.SupplierOperatorAddress),
		client:     client,
		config:     config,
		streamName: transport.StreamName(config.StreamPrefix, config.SupplierOperatorAddress),
		msgCh:      make(chan transport.StreamMessage, config.BatchSize*2),
	}, nil
}

// Consume returns a channel that yields mined relay messages.
func (c *StreamsConsumer) Consume(ctx context.Context) <-chan transport.StreamMessage {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		close(c.msgCh)
		return c.msgCh
	}

	// Create cancellable context
	ctx, c.cancelFn = context.WithCancel(ctx)
	c.mu.Unlock()

	// Ensure consumer group exists
	if err := c.ensureConsumerGroup(ctx); err != nil {
		c.logger.Error().Err(err).Msg("failed to ensure consumer group")
		close(c.msgCh)
		return c.msgCh
	}

	// Start consumer goroutine
	c.wg.Add(1)
	go c.consumeLoop(ctx)

	// Start pending message claimer (handles crashed consumers)
	c.wg.Add(1)
	go c.claimIdleMessages(ctx)

	return c.msgCh
}

// ensureConsumerGroup creates the consumer group if it doesn't exist.
func (c *StreamsConsumer) ensureConsumerGroup(ctx context.Context) error {
	// Try to create the consumer group
	// MKSTREAM creates the stream if it doesn't exist
	err := c.client.XGroupCreateMkStream(ctx, c.streamName, c.config.ConsumerGroup, "0").Err()
	if err != nil {
		// Ignore "BUSYGROUP" error - group already exists
		// Use strings.Contains for robustness against Redis version differences
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("failed to create consumer group: %w", err)
		}
	}

	c.logger.Info().
		Str(logging.FieldStreamID, c.streamName).
		Str("group", c.config.ConsumerGroup).
		Msg("consumer group ready")

	return nil
}

// consumeLoop is the main consumption loop.
func (c *StreamsConsumer) consumeLoop(ctx context.Context) {
	defer c.wg.Done()
	defer close(c.msgCh)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info().Msg("consume loop shutting down")
			return
		default:
		}

		// Read new messages from stream
		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.config.ConsumerGroup,
			Consumer: c.config.ConsumerName,
			Streams:  []string{c.streamName, ">"},
			Count:    c.config.BatchSize,
			Block:    time.Duration(c.config.BlockTimeout) * time.Millisecond,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				// No new messages, continue
				continue
			}
			if ctx.Err() != nil {
				// Context cancelled
				return
			}
			consumeErrorsTotal.WithLabelValues(c.config.SupplierOperatorAddress, "read_error").Inc()
			c.logger.Error().Err(err).Msg("error reading from stream")
			time.Sleep(time.Second) // Back off on error
			continue
		}

		// Process messages
		for _, stream := range streams {
			for _, message := range stream.Messages {
				msg, err := c.parseMessage(message)
				if err != nil {
					deserializationErrors.WithLabelValues(c.config.SupplierOperatorAddress).Inc()
					c.logger.Error().
						Err(err).
						Str(logging.FieldMessageID, message.ID).
						Msg("failed to parse message")
					// Acknowledge bad message to avoid redelivery
					_ = c.client.XAck(ctx, c.streamName, c.config.ConsumerGroup, message.ID)
					continue
				}

				// Record end-to-end latency
				if msg.Message.PublishedAtUnixNano > 0 {
					latency := time.Since(msg.Message.PublishedAt()).Seconds()
					endToEndLatency.WithLabelValues(
						c.config.SupplierOperatorAddress,
						msg.Message.ServiceId,
					).Observe(latency)
				}

				consumedTotal.WithLabelValues(
					c.config.SupplierOperatorAddress,
					msg.Message.ServiceId,
				).Inc()

				// Send to channel (blocks if channel is full)
				select {
				case c.msgCh <- *msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// claimIdleMessages periodically claims messages from idle consumers.
// This handles the case where a consumer crashes without acknowledging messages.
func (c *StreamsConsumer) claimIdleMessages(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Duration(c.config.ClaimIdleTimeout/2) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.claimPendingMessages(ctx)
		}
	}
}

// claimPendingMessages claims messages that have been pending too long.
func (c *StreamsConsumer) claimPendingMessages(ctx context.Context) {
	// Get pending messages that have been idle too long
	messages, _, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   c.streamName,
		Group:    c.config.ConsumerGroup,
		Consumer: c.config.ConsumerName,
		MinIdle:  time.Duration(c.config.ClaimIdleTimeout) * time.Millisecond,
		Start:    "0-0",
		Count:    c.config.BatchSize,
	}).Result()

	if err != nil {
		if ctx.Err() == nil {
			c.logger.Debug().Err(err).Msg("error claiming idle messages")
		}
		return
	}

	if len(messages) == 0 {
		return
	}

	claimedMessages.WithLabelValues(c.config.SupplierOperatorAddress).Add(float64(len(messages)))

	c.logger.Debug().
		Int("count", len(messages)).
		Msg("claimed idle messages")

	// Process claimed messages
	for _, message := range messages {
		msg, err := c.parseMessage(message)
		if err != nil {
			deserializationErrors.WithLabelValues(c.config.SupplierOperatorAddress).Inc()
			// Acknowledge bad message
			_ = c.client.XAck(ctx, c.streamName, c.config.ConsumerGroup, message.ID)
			continue
		}

		select {
		case c.msgCh <- *msg:
		case <-ctx.Done():
			return
		}
	}
}

// parseMessage deserializes a Redis Stream message into a StreamMessage.
func (c *StreamsConsumer) parseMessage(message redis.XMessage) (*transport.StreamMessage, error) {
	data, ok := message.Values["data"]
	if !ok {
		return nil, fmt.Errorf("message missing 'data' field")
	}

	dataStr, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("message 'data' field is not a string")
	}

	var minedRelay transport.MinedRelayMessage
	if err := json.Unmarshal([]byte(dataStr), &minedRelay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &transport.StreamMessage{
		ID:      message.ID,
		Message: &minedRelay,
	}, nil
}

// Ack acknowledges that a message has been successfully processed.
func (c *StreamsConsumer) Ack(ctx context.Context, messageID string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	err := c.client.XAck(ctx, c.streamName, c.config.ConsumerGroup, messageID).Err()
	if err != nil {
		return fmt.Errorf("failed to ack message %s: %w", messageID, err)
	}

	ackedTotal.WithLabelValues(c.config.SupplierOperatorAddress).Inc()
	return nil
}

// AckBatch acknowledges multiple messages in a single operation.
func (c *StreamsConsumer) AckBatch(ctx context.Context, messageIDs []string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	if len(messageIDs) == 0 {
		return nil
	}

	err := c.client.XAck(ctx, c.streamName, c.config.ConsumerGroup, messageIDs...).Err()
	if err != nil {
		return fmt.Errorf("failed to batch ack: %w", err)
	}

	ackedTotal.WithLabelValues(c.config.SupplierOperatorAddress).Add(float64(len(messageIDs)))
	return nil
}

// Pending returns the number of messages that have been delivered but not yet acknowledged.
func (c *StreamsConsumer) Pending(ctx context.Context) (int64, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return 0, fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	info, err := c.client.XPending(ctx, c.streamName, c.config.ConsumerGroup).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending info: %w", err)
	}

	pendingMessages.WithLabelValues(c.config.SupplierOperatorAddress).Set(float64(info.Count))
	return info.Count, nil
}

// Close gracefully shuts down the consumer.
func (c *StreamsConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Cancel context to stop goroutines
	if c.cancelFn != nil {
		c.cancelFn()
	}

	// Wait for goroutines to finish
	c.wg.Wait()

	c.logger.Info().Msg("Redis Streams consumer closed")
	return nil
}
