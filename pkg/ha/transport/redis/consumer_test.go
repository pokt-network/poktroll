package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func TestStreamsConsumer_Consume(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := polyzero.NewLogger()
	supplierAddr := "pokt1supplier123"
	streamPrefix := "test:relays"
	streamName := transport.StreamName(streamPrefix, supplierAddr)

	config := transport.ConsumerConfig{
		StreamPrefix:            streamPrefix,
		SupplierOperatorAddress: supplierAddr,
		ConsumerGroup:           "test-group",
		ConsumerName:            "test-consumer-1",
		BatchSize:               10,
		BlockTimeout:            100, // 100ms for faster tests
		ClaimIdleTimeout:        5000,
	}

	consumer, err := NewStreamsConsumer(logger, client, config)
	require.NoError(t, err)
	defer consumer.Close()

	// Publish test messages directly to the stream
	testMsg := &transport.MinedRelayMessage{
		RelayHash:               []byte("test-hash"),
		RelayBytes:              []byte("test-bytes"),
		ComputeUnitsPerRelay:    100,
		SessionId:               "session-123",
		SessionEndHeight:        1000,
		SupplierOperatorAddress: supplierAddr,
		ServiceId:               "ethereum",
		ApplicationAddress:      "pokt1app123",
		ArrivalBlockHeight:      999,
		PublishedAtUnixNano:     time.Now().UnixNano(),
	}

	data, err := json.Marshal(testMsg)
	require.NoError(t, err)

	_, err = client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{"data": string(data)},
	}).Result()
	require.NoError(t, err)

	// Start consuming
	msgCh := consumer.Consume(ctx)

	// Receive the message
	select {
	case msg, ok := <-msgCh:
		require.True(t, ok, "channel should not be closed")
		require.NotNil(t, msg.Message)
		require.Equal(t, testMsg.SessionId, msg.Message.SessionId)
		require.Equal(t, testMsg.ServiceId, msg.Message.ServiceId)
		require.Equal(t, testMsg.SupplierOperatorAddress, msg.Message.SupplierOperatorAddress)

		// Acknowledge the message
		err = consumer.Ack(ctx, msg.ID)
		require.NoError(t, err)

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestStreamsConsumer_Ack(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := polyzero.NewLogger()
	supplierAddr := "pokt1supplier123"
	streamPrefix := "test:relays"
	streamName := transport.StreamName(streamPrefix, supplierAddr)

	config := transport.ConsumerConfig{
		StreamPrefix:            streamPrefix,
		SupplierOperatorAddress: supplierAddr,
		ConsumerGroup:           "test-group",
		ConsumerName:            "test-consumer-1",
		BatchSize:               10,
		BlockTimeout:            100,
		ClaimIdleTimeout:        5000,
	}

	consumer, err := NewStreamsConsumer(logger, client, config)
	require.NoError(t, err)
	defer consumer.Close()

	// Publish a message
	testMsg := &transport.MinedRelayMessage{
		RelayHash:               []byte("hash"),
		SupplierOperatorAddress: supplierAddr,
		ServiceId:               "ethereum",
		PublishedAtUnixNano:     time.Now().UnixNano(),
	}
	data, _ := json.Marshal(testMsg)
	_, err = client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{"data": string(data)},
	}).Result()
	require.NoError(t, err)

	// Start consuming
	msgCh := consumer.Consume(ctx)

	// Receive message
	var msg transport.StreamMessage
	select {
	case msg = <-msgCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	// Check pending before ack
	pending, err := consumer.Pending(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), pending)

	// Ack the message
	err = consumer.Ack(ctx, msg.ID)
	require.NoError(t, err)

	// Check pending after ack
	pending, err = consumer.Pending(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), pending)
}

func TestStreamsConsumer_AckBatch(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := polyzero.NewLogger()
	supplierAddr := "pokt1supplier123"
	streamPrefix := "test:relays"
	streamName := transport.StreamName(streamPrefix, supplierAddr)

	config := transport.ConsumerConfig{
		StreamPrefix:            streamPrefix,
		SupplierOperatorAddress: supplierAddr,
		ConsumerGroup:           "test-group",
		ConsumerName:            "test-consumer-1",
		BatchSize:               10,
		BlockTimeout:            100,
		ClaimIdleTimeout:        5000,
	}

	consumer, err := NewStreamsConsumer(logger, client, config)
	require.NoError(t, err)
	defer consumer.Close()

	// Publish multiple messages
	for i := 0; i < 3; i++ {
		testMsg := &transport.MinedRelayMessage{
			RelayHash:               []byte{byte(i)},
			SupplierOperatorAddress: supplierAddr,
			ServiceId:               "ethereum",
			PublishedAtUnixNano:     time.Now().UnixNano(),
		}
		data, _ := json.Marshal(testMsg)
		_, err = client.XAdd(ctx, &redis.XAddArgs{
			Stream: streamName,
			Values: map[string]interface{}{"data": string(data)},
		}).Result()
		require.NoError(t, err)
	}

	// Start consuming
	msgCh := consumer.Consume(ctx)

	// Collect message IDs
	var messageIDs []string
	for i := 0; i < 3; i++ {
		select {
		case msg := <-msgCh:
			messageIDs = append(messageIDs, msg.ID)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}

	require.Len(t, messageIDs, 3)

	// Batch ack
	err = consumer.AckBatch(ctx, messageIDs)
	require.NoError(t, err)

	// Verify all acknowledged
	pending, err := consumer.Pending(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), pending)
}

func TestStreamsConsumer_ConfigValidation(t *testing.T) {
	_, client := setupTestRedis(t)
	logger := polyzero.NewLogger()

	tests := []struct {
		name        string
		config      transport.ConsumerConfig
		expectedErr string
	}{
		{
			name: "missing stream prefix",
			config: transport.ConsumerConfig{
				SupplierOperatorAddress: "addr",
				ConsumerGroup:           "group",
				ConsumerName:            "name",
			},
			expectedErr: "stream prefix is required",
		},
		{
			name: "missing supplier address",
			config: transport.ConsumerConfig{
				StreamPrefix:  "prefix",
				ConsumerGroup: "group",
				ConsumerName:  "name",
			},
			expectedErr: "supplier operator address is required",
		},
		{
			name: "missing consumer group",
			config: transport.ConsumerConfig{
				StreamPrefix:            "prefix",
				SupplierOperatorAddress: "addr",
				ConsumerName:            "name",
			},
			expectedErr: "consumer group is required",
		},
		{
			name: "missing consumer name",
			config: transport.ConsumerConfig{
				StreamPrefix:            "prefix",
				SupplierOperatorAddress: "addr",
				ConsumerGroup:           "group",
			},
			expectedErr: "consumer name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStreamsConsumer(logger, client, tt.config)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestStreamsConsumer_Close(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	logger := polyzero.NewLogger()

	config := transport.ConsumerConfig{
		StreamPrefix:            "test:relays",
		SupplierOperatorAddress: "pokt1supplier123",
		ConsumerGroup:           "test-group",
		ConsumerName:            "test-consumer-1",
		BatchSize:               10,
		BlockTimeout:            100,
		ClaimIdleTimeout:        5000,
	}

	consumer, err := NewStreamsConsumer(logger, client, config)
	require.NoError(t, err)

	// Start consuming
	msgCh := consumer.Consume(ctx)

	// Close the consumer
	err = consumer.Close()
	require.NoError(t, err)

	// Channel should be closed
	_, ok := <-msgCh
	require.False(t, ok, "channel should be closed")

	// Ack should fail on closed consumer
	err = consumer.Ack(ctx, "some-id")
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestStreamsConsumer_Defaults(t *testing.T) {
	_, client := setupTestRedis(t)
	logger := polyzero.NewLogger()

	config := transport.ConsumerConfig{
		StreamPrefix:            "test:relays",
		SupplierOperatorAddress: "pokt1supplier123",
		ConsumerGroup:           "test-group",
		ConsumerName:            "test-consumer-1",
		// Leave others at zero to test defaults
	}

	consumer, err := NewStreamsConsumer(logger, client, config)
	require.NoError(t, err)
	defer consumer.Close()

	// Verify defaults were applied
	require.Equal(t, int64(100), consumer.config.BatchSize)
	require.Equal(t, int64(5000), consumer.config.BlockTimeout)
	require.Equal(t, int64(30000), consumer.config.ClaimIdleTimeout)
	require.Equal(t, int64(3), consumer.config.MaxRetries)
}
