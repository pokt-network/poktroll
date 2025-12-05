package miner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	require.Equal(t, "ha:relays", config.Redis.StreamPrefix)
	require.Equal(t, "miner", config.Redis.ConsumerGroup)
	require.Equal(t, int64(5000), config.Redis.BlockTimeoutMs)
	require.Equal(t, int64(60000), config.Redis.ClaimIdleTimeoutMs)
	require.Equal(t, "badger", config.SessionTree.StorageType)
	require.True(t, config.SessionTree.WALEnabled)
	require.Equal(t, int64(10), config.DeduplicationTTLBlocks)
	require.Equal(t, int64(100), config.BatchSize)
	require.Equal(t, int64(50), config.AckBatchSize)
}

func TestConfig_Validate_Valid(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				OperatorAddress: "pokt1supplier123",
				SigningKeyName:  "supplier_key",
			},
		},
		SessionTree: SessionTreeConfig{
			StorageType: "badger",
			StoragePath: "/tmp/sessions",
		},
	}

	err := config.Validate()
	require.NoError(t, err)
}

func TestConfig_Validate_MissingRedisURL(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis.url is required")
}

func TestConfig_Validate_InvalidRedisURL(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "://invalid",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid redis.url")
}

func TestConfig_Validate_MissingStreamPrefix(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis.stream_prefix is required")
}

func TestConfig_Validate_MissingConsumerGroup(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:          "redis://localhost:6379",
			StreamPrefix: "test:relays",
			ConsumerName: "miner-1",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis.consumer_group is required")
}

func TestConfig_Validate_MissingConsumerName(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis.consumer_name is required")
}

func TestConfig_Validate_MissingPocketNodeRPC(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeGRPCUrl: "localhost:9090",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "pocket_node.query_node_rpc_url is required")
}

func TestConfig_Validate_MissingPocketNodeGRPC(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl: "http://localhost:26657",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "pocket_node.query_node_grpc_url is required")
}

func TestConfig_Validate_NoSuppliers(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one supplier must be configured")
}

func TestConfig_Validate_SupplierMissingAddress(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				SigningKeyName: "key1",
			},
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "suppliers[0].operator_address is required")
}

func TestConfig_Validate_SupplierMissingKeyName(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				OperatorAddress: "pokt1supplier123",
			},
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "suppliers[0].signing_key_name is required")
}

func TestConfig_Validate_InvalidStorageType(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				OperatorAddress: "pokt1supplier123",
				SigningKeyName:  "key1",
			},
		},
		SessionTree: SessionTreeConfig{
			StorageType: "invalid",
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid session_tree.storage_type")
}

func TestConfig_Validate_MissingStoragePath(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				OperatorAddress: "pokt1supplier123",
				SigningKeyName:  "key1",
			},
		},
		SessionTree: SessionTreeConfig{
			StorageType: "badger",
			// Missing StoragePath
		},
	}

	err := config.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "session_tree.storage_path is required")
}

func TestConfig_Validate_MemoryStorageNoPath(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				OperatorAddress: "pokt1supplier123",
				SigningKeyName:  "key1",
			},
		},
		SessionTree: SessionTreeConfig{
			StorageType: "memory",
			// No StoragePath needed for memory
		},
	}

	err := config.Validate()
	require.NoError(t, err)
}

func TestConfig_GetRedisBlockTimeout(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			BlockTimeoutMs: 10000,
		},
	}

	require.Equal(t, 10*time.Second, config.GetRedisBlockTimeout())

	// Test default
	config.Redis.BlockTimeoutMs = 0
	require.Equal(t, 5*time.Second, config.GetRedisBlockTimeout())
}

func TestConfig_GetClaimIdleTimeout(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			ClaimIdleTimeoutMs: 120000,
		},
	}

	require.Equal(t, 2*time.Minute, config.GetClaimIdleTimeout())

	// Test default
	config.Redis.ClaimIdleTimeoutMs = 0
	require.Equal(t, time.Minute, config.GetClaimIdleTimeout())
}

func TestConfig_GetBatchSize(t *testing.T) {
	config := &Config{BatchSize: 200}
	require.Equal(t, int64(200), config.GetBatchSize())

	// Test default
	config.BatchSize = 0
	require.Equal(t, int64(100), config.GetBatchSize())
}

func TestConfig_GetAckBatchSize(t *testing.T) {
	config := &Config{AckBatchSize: 25}
	require.Equal(t, int64(25), config.GetAckBatchSize())

	// Test default
	config.AckBatchSize = 0
	require.Equal(t, int64(50), config.GetAckBatchSize())
}

func TestConfig_GetDeduplicationTTL(t *testing.T) {
	config := &Config{DeduplicationTTLBlocks: 20}
	require.Equal(t, int64(20), config.GetDeduplicationTTL())

	// Test default
	config.DeduplicationTTLBlocks = 0
	require.Equal(t, int64(10), config.GetDeduplicationTTL())
}

func TestSupplierConfig_WithServices(t *testing.T) {
	supplier := SupplierConfig{
		OperatorAddress: "pokt1supplier123",
		SigningKeyName:  "key1",
		Services:        []string{"ethereum", "polygon", "anvil"},
	}

	require.Equal(t, "pokt1supplier123", supplier.OperatorAddress)
	require.Equal(t, "key1", supplier.SigningKeyName)
	require.Len(t, supplier.Services, 3)
	require.Contains(t, supplier.Services, "ethereum")
}

func TestConfig_Validate_MultipleSuppliers(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			StreamPrefix:  "test:relays",
			ConsumerGroup: "miner-group",
			ConsumerName:  "miner-1",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		Suppliers: []SupplierConfig{
			{
				OperatorAddress: "pokt1supplier1",
				SigningKeyName:  "key1",
				Services:        []string{"ethereum"},
			},
			{
				OperatorAddress: "pokt1supplier2",
				SigningKeyName:  "key2",
				Services:        []string{"polygon", "anvil"},
			},
		},
		SessionTree: SessionTreeConfig{
			StorageType: "memory",
		},
	}

	err := config.Validate()
	require.NoError(t, err)
}
