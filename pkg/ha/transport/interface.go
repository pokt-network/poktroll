package transport

import (
	"context"
)

// MinedRelayPublisher publishes mined relays to the transport layer.
// The Relayer service uses this interface to send mined relays to the Miner service.
//
// Implementations must be safe for concurrent use by multiple goroutines.
type MinedRelayPublisher interface {
	// Publish sends a mined relay message to the transport layer.
	// The message is routed based on the SupplierOperatorAddress.
	//
	// This operation is fire-and-forget with acknowledgment:
	// - Returns nil on successful publish (message accepted by transport)
	// - Returns error on failure (network error, serialization error, etc.)
	//
	// The publisher should handle transient failures with internal retries.
	Publish(ctx context.Context, msg *MinedRelayMessage) error

	// PublishBatch sends multiple mined relay messages in a single operation.
	// More efficient than individual Publish calls for high throughput scenarios.
	//
	// All messages in the batch should be for the same supplier (same stream).
	// Returns error if any message fails to publish.
	PublishBatch(ctx context.Context, msgs []*MinedRelayMessage) error

	// Close gracefully shuts down the publisher, flushing any buffered messages.
	Close() error
}

// MinedRelayConsumer consumes mined relays from the transport layer.
// The Miner service uses this interface to receive mined relays from Relayer instances.
//
// Implementations must provide exactly-once delivery semantics within the consumer group.
type MinedRelayConsumer interface {
	// Consume returns a channel that yields mined relay messages.
	// Messages are not acknowledged until Ack is called.
	//
	// The channel is closed when:
	// - The context is cancelled
	// - Close() is called
	// - An unrecoverable error occurs
	//
	// Callers should handle channel closure gracefully.
	Consume(ctx context.Context) <-chan StreamMessage

	// Ack acknowledges that a message has been successfully processed.
	// The message will not be redelivered to any consumer in the group.
	//
	// Call this AFTER the relay has been:
	// 1. Deduplicated
	// 2. Added to the session tree
	// 3. Persisted to WAL (if applicable)
	Ack(ctx context.Context, messageID string) error

	// AckBatch acknowledges multiple messages in a single operation.
	// More efficient than individual Ack calls.
	AckBatch(ctx context.Context, messageIDs []string) error

	// Pending returns the number of messages that have been delivered but not yet acknowledged.
	// Useful for monitoring consumer health and backpressure.
	Pending(ctx context.Context) (int64, error)

	// Close gracefully shuts down the consumer.
	// Any unacknowledged messages will be redelivered to other consumers in the group.
	Close() error
}

// ConsumerConfig contains configuration for a MinedRelayConsumer.
type ConsumerConfig struct {
	// StreamPrefix is the prefix for Redis stream names.
	// Full stream name: {StreamPrefix}:{SupplierOperatorAddress}
	StreamPrefix string

	// SupplierOperatorAddress is the supplier this consumer reads relays for.
	SupplierOperatorAddress string

	// ConsumerGroup is the Redis consumer group name.
	// All Miner instances for the same supplier should use the same group.
	ConsumerGroup string

	// ConsumerName is the unique name of this consumer within the group.
	// Typically includes hostname/pod name for identification.
	ConsumerName string

	// BatchSize is the maximum number of messages to fetch per read operation.
	BatchSize int64

	// BlockTimeout is how long to wait for new messages when the stream is empty.
	// Set to 0 for non-blocking reads.
	BlockTimeout int64

	// MaxRetries is the maximum number of times to retry a failed message.
	// After this, the message is moved to the dead letter queue.
	MaxRetries int64

	// ClaimIdleTimeout is how long a message can be pending before being claimed
	// by another consumer. This handles consumer crashes.
	ClaimIdleTimeout int64
}

// PublisherConfig contains configuration for a MinedRelayPublisher.
type PublisherConfig struct {
	// StreamPrefix is the prefix for Redis stream names.
	// Full stream name: {StreamPrefix}:{SupplierOperatorAddress}
	StreamPrefix string

	// MaxLen is the maximum number of messages in each stream.
	// Older messages are trimmed when this limit is exceeded.
	// Set to 0 for no limit (not recommended in production).
	MaxLen int64

	// ApproxMaxLen uses approximate trimming for better performance.
	// Recommended for high-throughput scenarios.
	ApproxMaxLen bool
}

// StreamName returns the full Redis stream name for a supplier.
func StreamName(prefix, supplierOperatorAddress string) string {
	return prefix + ":" + supplierOperatorAddress
}
