package miner

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the configuration for the HA Miner service.
type Config struct {
	// Redis configuration for consuming mined relays.
	Redis RedisConfig `yaml:"redis"`

	// PocketNode is the configuration for connecting to the Pocket blockchain.
	PocketNode PocketNodeConfig `yaml:"pocket_node"`

	// Keys configuration for loading supplier signing keys.
	Keys KeysConfig `yaml:"keys"`

	// Suppliers is a list of supplier configurations this miner manages.
	// Each supplier has its own session trees, claims, and proofs.
	Suppliers []SupplierConfig `yaml:"suppliers"`

	// SessionTree configuration for SMST management.
	SessionTree SessionTreeConfig `yaml:"session_tree"`

	// Metrics configuration.
	Metrics MetricsConfig `yaml:"metrics"`

	// Logging configuration.
	Logging LoggingConfig `yaml:"logging"`

	// DeduplicationTTLBlocks is how many blocks to keep relay hashes for deduplication.
	// Default: 10 (session length + grace period + buffer)
	DeduplicationTTLBlocks int64 `yaml:"deduplication_ttl_blocks"`

	// BatchSize is the number of relays to process in a single batch.
	// Default: 100
	BatchSize int64 `yaml:"batch_size"`

	// AckBatchSize is the number of messages to acknowledge in a batch.
	// Default: 50
	AckBatchSize int64 `yaml:"ack_batch_size"`

	// HotReloadEnabled enables hot-reload of keys.
	// Default: true
	HotReloadEnabled bool `yaml:"hot_reload_enabled"`

	// SessionTTL is the TTL for session data.
	// Default: 24h
	SessionTTL time.Duration `yaml:"session_ttl"`

	// WALMaxLen is the maximum WAL entries per session.
	// Default: 100000
	WALMaxLen int64 `yaml:"wal_max_len"`
}

// KeysConfig contains key provider configuration.
type KeysConfig struct {
	// KeysFile is the path to a supplier.yaml file with hex-encoded keys.
	KeysFile string `yaml:"keys_file,omitempty"`

	// KeysDir is a directory containing individual key files.
	KeysDir string `yaml:"keys_dir,omitempty"`

	// Keyring configuration for Cosmos SDK keyring.
	Keyring *KeyringConfig `yaml:"keyring,omitempty"`
}

// KeyringConfig contains Cosmos SDK keyring configuration.
type KeyringConfig struct {
	// Backend is the keyring backend type: "file", "os", "test", "memory"
	Backend string `yaml:"backend"`

	// Dir is the directory containing the keyring (for "file" backend).
	Dir string `yaml:"dir,omitempty"`

	// AppName is the application name for the keyring.
	// Default: "pocket"
	AppName string `yaml:"app_name,omitempty"`

	// KeyNames is an optional list of specific key names to load.
	KeyNames []string `yaml:"key_names,omitempty"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	// Level is the log level: "trace", "debug", "info", "warn", "error"
	// Default: "info"
	Level string `yaml:"level"`

	// Format is the log format: "json" or "text"
	// Default: "json"
	Format string `yaml:"format"`
}

// RedisConfig contains Redis connection configuration.
type RedisConfig struct {
	// URL is the Redis connection URL.
	URL string `yaml:"url"`

	// StreamPrefix is the prefix for Redis stream names.
	StreamPrefix string `yaml:"stream_prefix"`

	// ConsumerGroup is the consumer group name for this miner cluster.
	// All miner instances for the same supplier should use the same group.
	ConsumerGroup string `yaml:"consumer_group"`

	// ConsumerName is the unique name of this miner instance.
	// Typically derived from hostname/pod name.
	ConsumerName string `yaml:"consumer_name"`

	// BlockTimeout is how long to wait for new messages (milliseconds).
	// Default: 5000 (5 seconds)
	BlockTimeoutMs int64 `yaml:"block_timeout_ms"`

	// ClaimIdleTimeoutMs is how long a message can be pending before being claimed.
	// Default: 60000 (1 minute)
	ClaimIdleTimeoutMs int64 `yaml:"claim_idle_timeout_ms"`
}

// PocketNodeConfig contains Pocket blockchain connection configuration.
type PocketNodeConfig struct {
	// QueryNodeRPCUrl is the URL for RPC queries.
	QueryNodeRPCUrl string `yaml:"query_node_rpc_url"`

	// QueryNodeGRPCUrl is the URL for gRPC queries.
	QueryNodeGRPCUrl string `yaml:"query_node_grpc_url"`

	// TxNodeRPCUrl is the URL for transaction submission (if different from query).
	TxNodeRPCUrl string `yaml:"tx_node_rpc_url,omitempty"`

	// GRPCInsecure disables TLS for gRPC connections.
	// Default: false (use TLS for secure connections)
	GRPCInsecure bool `yaml:"grpc_insecure,omitempty"`
}

// SupplierConfig contains configuration for a single supplier.
type SupplierConfig struct {
	// OperatorAddress is the supplier's operator address (bech32).
	OperatorAddress string `yaml:"operator_address"`

	// SigningKeyName is the name of the key in the keyring used for signing.
	SigningKeyName string `yaml:"signing_key_name"`

	// Services is a list of service IDs this supplier serves.
	// Used for filtering relays from the stream.
	Services []string `yaml:"services,omitempty"`
}

// SessionTreeConfig contains configuration for session tree management.
type SessionTreeConfig struct {
	// StorageType is the type of storage for session trees.
	// Options: "memory", "badger", "pebble"
	// Default: "badger"
	StorageType string `yaml:"storage_type"`

	// StoragePath is the path for persistent storage.
	// Required for "badger" and "pebble" storage types.
	StoragePath string `yaml:"storage_path"`

	// WALEnabled enables Write-Ahead Log for crash recovery.
	// Default: true
	WALEnabled bool `yaml:"wal_enabled"`

	// WALPath is the path for WAL files.
	// Default: {StoragePath}/wal
	WALPath string `yaml:"wal_path,omitempty"`

	// FlushInterval is how often to flush session trees (in blocks).
	// Default: 1
	FlushIntervalBlocks int64 `yaml:"flush_interval_blocks"`
}

// MetricsConfig contains Prometheus metrics configuration.
type MetricsConfig struct {
	// Enabled enables metrics collection.
	Enabled bool `yaml:"enabled"`

	// Addr is the address to expose metrics on.
	// Default: ":9091"
	Addr string `yaml:"addr"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Redis.URL == "" {
		return fmt.Errorf("redis.url is required")
	}

	if _, err := url.Parse(c.Redis.URL); err != nil {
		return fmt.Errorf("invalid redis.url: %w", err)
	}

	if c.Redis.StreamPrefix == "" {
		return fmt.Errorf("redis.stream_prefix is required")
	}

	if c.Redis.ConsumerGroup == "" {
		return fmt.Errorf("redis.consumer_group is required")
	}

	if c.Redis.ConsumerName == "" {
		return fmt.Errorf("redis.consumer_name is required")
	}

	if c.PocketNode.QueryNodeRPCUrl == "" {
		return fmt.Errorf("pocket_node.query_node_rpc_url is required")
	}

	if c.PocketNode.QueryNodeGRPCUrl == "" {
		return fmt.Errorf("pocket_node.query_node_grpc_url is required")
	}

	// Either keys config or explicit suppliers must be configured
	if !c.HasKeySource() && len(c.Suppliers) == 0 {
		return fmt.Errorf("either keys config or at least one supplier must be configured")
	}

	// Validate explicit suppliers if configured
	for i, supplier := range c.Suppliers {
		if err := c.validateSupplierConfig(i, supplier); err != nil {
			return err
		}
	}

	// Validate keyring config if provided
	if c.Keys.Keyring != nil && c.Keys.Keyring.Backend != "" {
		validBackends := map[string]bool{"file": true, "os": true, "test": true, "memory": true}
		if !validBackends[c.Keys.Keyring.Backend] {
			return fmt.Errorf("invalid keys.keyring.backend: %s", c.Keys.Keyring.Backend)
		}
	}

	if c.SessionTree.StorageType != "" &&
		c.SessionTree.StorageType != "memory" &&
		c.SessionTree.StorageType != "badger" &&
		c.SessionTree.StorageType != "pebble" {
		return fmt.Errorf("invalid session_tree.storage_type: %s", c.SessionTree.StorageType)
	}

	if c.SessionTree.StorageType != "memory" && c.SessionTree.StoragePath == "" {
		return fmt.Errorf("session_tree.storage_path is required for non-memory storage")
	}

	return nil
}

// validateSupplierConfig validates a single supplier configuration.
func (c *Config) validateSupplierConfig(index int, supplier SupplierConfig) error {
	if supplier.OperatorAddress == "" {
		return fmt.Errorf("suppliers[%d].operator_address is required", index)
	}

	if supplier.SigningKeyName == "" {
		return fmt.Errorf("suppliers[%d].signing_key_name is required", index)
	}

	return nil
}

// GetRedisBlockTimeout returns the Redis block timeout as a duration.
func (c *Config) GetRedisBlockTimeout() time.Duration {
	if c.Redis.BlockTimeoutMs > 0 {
		return time.Duration(c.Redis.BlockTimeoutMs) * time.Millisecond
	}
	return 5 * time.Second // Default
}

// GetClaimIdleTimeout returns the claim idle timeout as a duration.
func (c *Config) GetClaimIdleTimeout() time.Duration {
	if c.Redis.ClaimIdleTimeoutMs > 0 {
		return time.Duration(c.Redis.ClaimIdleTimeoutMs) * time.Millisecond
	}
	return time.Minute // Default
}

// GetBatchSize returns the batch size with defaults.
func (c *Config) GetBatchSize() int64 {
	if c.BatchSize > 0 {
		return c.BatchSize
	}
	return 100 // Default
}

// GetAckBatchSize returns the ack batch size with defaults.
func (c *Config) GetAckBatchSize() int64 {
	if c.AckBatchSize > 0 {
		return c.AckBatchSize
	}
	return 50 // Default
}

// GetDeduplicationTTL returns the deduplication TTL in blocks.
func (c *Config) GetDeduplicationTTL() int64 {
	if c.DeduplicationTTLBlocks > 0 {
		return c.DeduplicationTTLBlocks
	}
	return 10 // Default (session length + grace + buffer)
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Redis: RedisConfig{
			URL:                "redis://localhost:6379",
			StreamPrefix:       "ha:relays",
			ConsumerGroup:      "ha-miners",
			BlockTimeoutMs:     5000,
			ClaimIdleTimeoutMs: 60000,
		},
		SessionTree: SessionTreeConfig{
			StorageType:         "badger",
			StoragePath:         "./data/session-trees",
			WALEnabled:          true,
			WALPath:             "./data/wal",
			FlushIntervalBlocks: 1,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Addr:    ":9092",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		DeduplicationTTLBlocks: 10,
		BatchSize:              100,
		AckBatchSize:           50,
		HotReloadEnabled:       true,
		SessionTTL:             24 * time.Hour,
		WALMaxLen:              100000,
	}
}

// LoadConfig loads a miner configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with defaults
	config := DefaultConfig()

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Generate consumer name from hostname if not set
	if config.Redis.ConsumerName == "" {
		hostname, _ := os.Hostname()
		config.Redis.ConsumerName = fmt.Sprintf("miner-%s-%d", hostname, os.Getpid())
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// HasKeySource returns true if at least one key source is configured.
func (c *Config) HasKeySource() bool {
	return c.Keys.KeysFile != "" ||
		c.Keys.KeysDir != "" ||
		(c.Keys.Keyring != nil && c.Keys.Keyring.Backend != "")
}
