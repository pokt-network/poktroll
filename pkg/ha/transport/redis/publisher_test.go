package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, client
}

func TestStreamsPublisher_Publish(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
		MaxLen:       1000,
		ApproxMaxLen: true,
	}

	publisher := NewStreamsPublisher(logger, client, config)
	defer publisher.Close()

	msg := &transport.MinedRelayMessage{
		RelayHash:               []byte("test-hash-123"),
		RelayBytes:              []byte("test-relay-bytes"),
		ComputeUnitsPerRelay:    100,
		SessionId:               "session-123",
		SessionEndHeight:        1000,
		SupplierOperatorAddress: "pokt1supplier123",
		ServiceId:               "ethereum",
		ApplicationAddress:      "pokt1app123",
		ArrivalBlockHeight:      999,
	}

	// Publish the message
	err := publisher.Publish(ctx, msg)
	require.NoError(t, err)

	// Verify the message was published
	streamName := transport.StreamName(config.StreamPrefix, msg.SupplierOperatorAddress)
	streamLen, err := client.XLen(ctx, streamName).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), streamLen)

	// Read the message back
	messages, err := client.XRange(ctx, streamName, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	// Verify data field exists
	_, ok := messages[0].Values["data"]
	require.True(t, ok, "message should have 'data' field")
}

func TestStreamsPublisher_PublishBatch(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
		MaxLen:       1000,
	}

	publisher := NewStreamsPublisher(logger, client, config)
	defer publisher.Close()

	// Create multiple messages for the same supplier
	msgs := []*transport.MinedRelayMessage{
		{
			RelayHash:               []byte("hash-1"),
			RelayBytes:              []byte("bytes-1"),
			SessionId:               "session-1",
			SupplierOperatorAddress: "pokt1supplier123",
			ServiceId:               "ethereum",
		},
		{
			RelayHash:               []byte("hash-2"),
			RelayBytes:              []byte("bytes-2"),
			SessionId:               "session-1",
			SupplierOperatorAddress: "pokt1supplier123",
			ServiceId:               "ethereum",
		},
		{
			RelayHash:               []byte("hash-3"),
			RelayBytes:              []byte("bytes-3"),
			SessionId:               "session-2",
			SupplierOperatorAddress: "pokt1supplier123",
			ServiceId:               "anvil",
		},
	}

	// Publish batch
	err := publisher.PublishBatch(ctx, msgs)
	require.NoError(t, err)

	// Verify all messages were published
	streamName := transport.StreamName(config.StreamPrefix, "pokt1supplier123")
	streamLen, err := client.XLen(ctx, streamName).Result()
	require.NoError(t, err)
	require.Equal(t, int64(3), streamLen)
}

func TestStreamsPublisher_PublishBatch_MultipleSuppliers(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
	}

	publisher := NewStreamsPublisher(logger, client, config)
	defer publisher.Close()

	// Create messages for different suppliers
	msgs := []*transport.MinedRelayMessage{
		{
			RelayHash:               []byte("hash-1"),
			SupplierOperatorAddress: "pokt1supplierA",
			ServiceId:               "ethereum",
		},
		{
			RelayHash:               []byte("hash-2"),
			SupplierOperatorAddress: "pokt1supplierB",
			ServiceId:               "ethereum",
		},
		{
			RelayHash:               []byte("hash-3"),
			SupplierOperatorAddress: "pokt1supplierA",
			ServiceId:               "anvil",
		},
	}

	err := publisher.PublishBatch(ctx, msgs)
	require.NoError(t, err)

	// Verify messages in supplier A's stream
	streamA := transport.StreamName(config.StreamPrefix, "pokt1supplierA")
	lenA, err := client.XLen(ctx, streamA).Result()
	require.NoError(t, err)
	require.Equal(t, int64(2), lenA)

	// Verify messages in supplier B's stream
	streamB := transport.StreamName(config.StreamPrefix, "pokt1supplierB")
	lenB, err := client.XLen(ctx, streamB).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), lenB)
}

func TestStreamsPublisher_Publish_NilMessage(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
	}

	publisher := NewStreamsPublisher(logger, client, config)
	defer publisher.Close()

	err := publisher.Publish(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "message is nil")
}

func TestStreamsPublisher_Publish_Closed(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
	}

	publisher := NewStreamsPublisher(logger, client, config)

	// Close the publisher
	err := publisher.Close()
	require.NoError(t, err)

	// Try to publish
	msg := &transport.MinedRelayMessage{
		RelayHash:               []byte("hash"),
		SupplierOperatorAddress: "pokt1supplier",
	}
	err = publisher.Publish(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestStreamsPublisher_Publish_SetsTimestamp(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
	}

	publisher := NewStreamsPublisher(logger, client, config)
	defer publisher.Close()

	before := time.Now()

	msg := &transport.MinedRelayMessage{
		RelayHash:               []byte("hash"),
		SupplierOperatorAddress: "pokt1supplier",
		ServiceId:               "ethereum",
	}

	err := publisher.Publish(ctx, msg)
	require.NoError(t, err)

	after := time.Now()

	// Verify timestamp was set
	require.NotZero(t, msg.PublishedAtUnixNano)
	publishedAt := msg.PublishedAt()
	require.True(t, publishedAt.After(before) || publishedAt.Equal(before))
	require.True(t, publishedAt.Before(after) || publishedAt.Equal(after))
}

func TestStreamsPublisher_MaxLen_Trimming(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.PublisherConfig{
		StreamPrefix: "test:relays",
		MaxLen:       5, // Small limit for testing
		ApproxMaxLen: false,
	}

	publisher := NewStreamsPublisher(logger, client, config)
	defer publisher.Close()

	// Publish more messages than MaxLen
	for i := 0; i < 10; i++ {
		msg := &transport.MinedRelayMessage{
			RelayHash:               []byte{byte(i)},
			SupplierOperatorAddress: "pokt1supplier",
			ServiceId:               "ethereum",
		}
		err := publisher.Publish(ctx, msg)
		require.NoError(t, err)
	}

	// Fast forward miniredis to process MAXLEN
	mr.FastForward(time.Second)

	// Verify stream was trimmed
	streamName := transport.StreamName(config.StreamPrefix, "pokt1supplier")
	streamLen, err := client.XLen(ctx, streamName).Result()
	require.NoError(t, err)
	require.LessOrEqual(t, streamLen, int64(5))
}
