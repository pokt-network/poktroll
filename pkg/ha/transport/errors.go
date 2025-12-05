package transport

import (
	"errors"
)

var (
	// ErrPublisherClosed is returned when attempting to publish to a closed publisher.
	ErrPublisherClosed = errors.New("publisher is closed")

	// ErrConsumerClosed is returned when attempting to consume from a closed consumer.
	ErrConsumerClosed = errors.New("consumer is closed")

	// ErrMessageNil is returned when attempting to publish a nil message.
	ErrMessageNil = errors.New("message is nil")

	// ErrSerializationFailed is returned when message serialization fails.
	ErrSerializationFailed = errors.New("failed to serialize message")

	// ErrDeserializationFailed is returned when message deserialization fails.
	ErrDeserializationFailed = errors.New("failed to deserialize message")

	// ErrStreamNotFound is returned when the Redis stream doesn't exist.
	ErrStreamNotFound = errors.New("stream not found")

	// ErrConsumerGroupNotFound is returned when the consumer group doesn't exist.
	ErrConsumerGroupNotFound = errors.New("consumer group not found")

	// ErrAcknowledgeFailed is returned when message acknowledgment fails.
	ErrAcknowledgeFailed = errors.New("failed to acknowledge message")

	// ErrPublishFailed is returned when publishing a message fails.
	ErrPublishFailed = errors.New("failed to publish message")

	// ErrConfigInvalid is returned when the configuration is invalid.
	ErrConfigInvalid = errors.New("invalid configuration")
)
