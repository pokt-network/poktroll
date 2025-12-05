package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/ha/cache"
	haclient "github.com/pokt-network/poktroll/pkg/ha/client"
	"github.com/pokt-network/poktroll/pkg/ha/keys"
	"github.com/pokt-network/poktroll/pkg/ha/miner"
	"github.com/pokt-network/poktroll/pkg/ha/observability"
	"github.com/pokt-network/poktroll/pkg/ha/query"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/ha/tx"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

const (
	flagMinerConfig    = "config"
	flagKeysFile       = "keys-file"
	flagKeysDir        = "keys-dir"
	flagKeyringBackend = "keyring-backend"
	flagKeyringDir     = "keyring-dir"
	flagConsumerGroup  = "consumer-group"
	flagConsumerName   = "consumer-name"
	flagStreamPrefix   = "stream-prefix"
	flagHotReload      = "hot-reload"
	flagSessionTTL     = "session-ttl"
	flagWALMaxLen      = "wal-max-len"
)

// startMinerCmd returns the command for starting the HA Miner component.
func startMinerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "miner",
		Short: "Start the HA Miner (SMST builder and claim/proof submitter)",
		Long: `Start the High-Availability Miner component.

The HA Miner consumes mined relays from Redis Streams and builds SMST trees.
It supports multiple suppliers and dynamically adds/removes them based on key changes.

Configuration:
  --config: Path to miner config YAML file (recommended)

Legacy Key Sources (if not using config file):
  --keys-file: Path to supplier.yaml containing hex-encoded private keys
  --keys-dir: Directory containing individual key files (YAML/JSON)
  --keyring-backend/--keyring-dir: Cosmos keyring integration

Features:
- Multi-supplier support (one consumer per supplier)
- Consumes mined relays from Redis Streams
- Builds SMST (Sparse Merkle Sum Tree) for each session
- WAL-based crash recovery
- Hot-reload of keys (add/remove suppliers without restart)
- Publishes supplier registry for relayer discovery
- Prometheus metrics at /metrics

Example:
  pocketd relayminer ha miner --config /path/to/miner-config.yaml
  pocketd relayminer ha miner --keys-file /path/to/supplier.yaml --redis-url redis://localhost:6379
`,
		RunE: runHAMiner,
	}

	// Config file (recommended approach)
	cmd.Flags().String(flagMinerConfig, "", "Path to miner config YAML file")

	// Legacy key source flags (for backwards compatibility)
	cmd.Flags().String(flagKeysFile, "", "Path to supplier.yaml with hex-encoded private keys")
	cmd.Flags().String(flagKeysDir, "", "Directory containing individual key files (YAML/JSON)")
	cmd.Flags().String(flagKeyringBackend, "", "Cosmos keyring backend: file, os, test")
	cmd.Flags().String(flagKeyringDir, "", "Cosmos keyring directory")

	// Redis flags (can override config)
	cmd.Flags().String(flagRedisURL, "", "Redis connection URL (overrides config)")
	cmd.Flags().String(flagConsumerGroup, "", "Redis consumer group name (overrides config)")
	cmd.Flags().String(flagConsumerName, "", "Consumer name (defaults to hostname)")
	cmd.Flags().String(flagStreamPrefix, "", "Redis stream name prefix (overrides config)")

	// Configuration flags (can override config)
	cmd.Flags().Bool(flagHotReload, true, "Enable hot-reload of keys")
	cmd.Flags().Duration(flagSessionTTL, 24*time.Hour, "Session data TTL")
	cmd.Flags().Int64(flagWALMaxLen, 100000, "Maximum WAL entries per session")

	return cmd
}

func runHAMiner(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Set up logger
	logger := polyzero.NewLogger()

	// Load config - either from file or build from flags
	config, err := loadMinerConfig(cmd)
	if err != nil {
		return err
	}

	// Start observability server (metrics and pprof)
	if config.Metrics.Enabled {
		obsServer := observability.NewServer(logger, observability.ServerConfig{
			MetricsEnabled: config.Metrics.Enabled,
			MetricsAddr:    config.Metrics.Addr,
			PprofEnabled:   false, // pprof not yet configurable for miner
			Registry:       observability.MinerRegistry,
		})
		if err := obsServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start observability server: %w", err)
		}
		defer obsServer.Stop()
		logger.Info().Str("addr", config.Metrics.Addr).Msg("observability server started")
	}

	// Parse Redis URL
	redisOpts, err := redis.ParseURL(config.Redis.URL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.Info().
		Str("redis_url", config.Redis.URL).
		Str("consumer_name", config.Redis.ConsumerName).
		Msg("connected to Redis")

	// Create key providers from config
	providers, err := createKeyProviders(logger, config)
	if err != nil {
		return err
	}

	if len(providers) == 0 {
		return fmt.Errorf("no key providers configured")
	}

	// Create key manager
	keyManager := keys.NewMultiProviderKeyManager(
		logger,
		providers,
		keys.KeyManagerConfig{
			HotReloadEnabled: config.HotReloadEnabled,
		},
	)

	// Start key manager
	if err := keyManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start key manager: %w", err)
	}
	defer keyManager.Close()

	// Check if any keys were loaded
	suppliers := keyManager.ListSuppliers()
	if len(suppliers) == 0 {
		logger.Warn().Msg("no supplier keys found - miner will wait for keys to be added")
	} else {
		logger.Info().
			Int("count", len(suppliers)).
			Msg("loaded supplier keys")
	}

	// Create supplier registry
	registry := miner.NewSupplierRegistry(
		logger,
		redisClient,
		miner.SupplierRegistryConfig{
			KeyPrefix:    "ha:suppliers",
			IndexKey:     "ha:suppliers:index",
			EventChannel: "ha:events:supplier_update",
		},
	)

	// Create supplier cache for publishing supplier state to relayers
	supplierCache := cache.NewSupplierCache(
		logger,
		redisClient,
		cache.SupplierCacheConfig{
			KeyPrefix: "ha:supplier",
			FailOpen:  false, // Miner should fail-closed for writes
		},
	)
	logger.Info().Msg("supplier cache initialized for state publishing")

	// Create query clients to query supplier information from the blockchain
	queryClients, err := query.NewQueryClients(
		logger,
		query.QueryClientConfig{
			GRPCEndpoint: config.PocketNode.QueryNodeGRPCUrl,
			QueryTimeout: 30 * time.Second,
			UseTLS:       !config.PocketNode.GRPCInsecure,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create query clients: %w", err)
	}
	defer queryClients.Close()
	logger.Info().Str("grpc_endpoint", config.PocketNode.QueryNodeGRPCUrl).Msg("query clients initialized")

	// Create block poller for monitoring block heights (needed for claim/proof timing)
	blockPoller, err := haclient.NewBlockPoller(
		logger,
		haclient.BlockPollerConfig{
			RPCEndpoint:  config.PocketNode.QueryNodeRPCUrl,
			PollInterval: 1 * time.Second,
			UseTLS:       !config.PocketNode.GRPCInsecure,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create block poller: %w", err)
	}
	if err := blockPoller.Start(ctx); err != nil {
		return fmt.Errorf("failed to start block poller: %w", err)
	}
	defer blockPoller.Close()
	logger.Info().Str("rpc_endpoint", config.PocketNode.QueryNodeRPCUrl).Msg("block poller started")

	// Fetch chain ID from the node
	chainID, err := blockPoller.GetChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID from node: %w", err)
	}
	logger.Info().Str("chain_id", chainID).Msg("fetched chain ID from node")

	// Create transaction client for submitting claims and proofs
	// Reuse the gRPC connection from QueryClients to avoid creating a duplicate connection
	txClient, err := tx.NewTxClient(
		logger,
		keyManager,
		tx.TxClientConfig{
			GRPCConn:      queryClients.GRPCConnection(), // Share gRPC connection with QueryClients
			ChainID:       chainID,
			GasLimit:      tx.DefaultGasLimit,
			TimeoutBlocks: tx.DefaultTimeoutHeight,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction client: %w", err)
	}
	defer txClient.Close()
	logger.Info().Str("grpc_endpoint", config.PocketNode.QueryNodeGRPCUrl).Msg("transaction client initialized")

	// Create supplier manager
	supplierManager := miner.NewSupplierManager(
		logger,
		keyManager,
		registry,
		miner.SupplierManagerConfig{
			RedisClient:         redisClient,
			StreamPrefix:        config.Redis.StreamPrefix,
			ConsumerGroup:       config.Redis.ConsumerGroup,
			ConsumerName:        config.Redis.ConsumerName,
			SessionTTL:          config.SessionTTL,
			WALMaxLen:           config.WALMaxLen,
			SupplierCache:       supplierCache,
			MinerID:             config.Redis.ConsumerName,
			SupplierQueryClient: queryClients.Supplier(),
			// New clients for claim/proof lifecycle management
			TxClient:      txClient,
			BlockClient:   blockPoller,
			SharedClient:  queryClients.Shared(),
			SessionClient: queryClients.Session(),
		},
	)

	// Set relay handler
	supplierManager.SetRelayHandler(func(ctx context.Context, supplierAddr string, msg *transport.StreamMessage) error {
		state, ok := supplierManager.GetSupplierState(supplierAddr)
		if !ok {
			return fmt.Errorf("supplier state not found: %s", supplierAddr)
		}

		// Use the full metadata method to create session if it doesn't exist
		// This updates the WAL and session snapshot in Redis
		if err := state.SnapshotManager.OnRelayMinedWithMetadata(
			ctx,
			msg.Message.SessionId,
			msg.Message.RelayHash,
			msg.Message.RelayBytes,
			msg.Message.ComputeUnitsPerRelay,
			msg.Message.SupplierOperatorAddress,
			msg.Message.ServiceId,
			msg.Message.ApplicationAddress,
			msg.Message.SessionStartHeight,
			msg.Message.SessionEndHeight,
		); err != nil {
			return fmt.Errorf("failed to update snapshot manager: %w", err)
		}

		// Also update the in-memory SMST for claim/proof generation
		// This is critical - without this, FlushTree will fail with "session not found"
		return state.SMSTManager.UpdateTree(
			ctx,
			msg.Message.SessionId,
			msg.Message.RelayHash,
			msg.Message.RelayBytes,
			msg.Message.ComputeUnitsPerRelay,
		)
	})

	// Start supplier manager
	if err := supplierManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start supplier manager: %w", err)
	}
	defer supplierManager.Close()

	logger.Info().
		Int("suppliers", len(supplierManager.ListSuppliers())).
		Str("consumer_group", config.Redis.ConsumerGroup).
		Str("consumer_name", config.Redis.ConsumerName).
		Bool("hot_reload", config.HotReloadEnabled).
		Msg("HA Miner started")

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigCh
	logger.Info().Msg("shutdown signal received, stopping HA Miner...")

	// Graceful shutdown is handled by defers
	logger.Info().Msg("HA Miner stopped")
	return nil
}

// loadMinerConfig loads the miner configuration from file or flags.
func loadMinerConfig(cmd *cobra.Command) (*miner.Config, error) {
	configPath, _ := cmd.Flags().GetString(flagMinerConfig)

	var config *miner.Config
	var err error

	if configPath != "" {
		// Load from config file
		config, err = miner.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		// Build config from flags (legacy mode)
		config = miner.DefaultConfig()

		// Key sources
		keysFile, _ := cmd.Flags().GetString(flagKeysFile)
		keysDir, _ := cmd.Flags().GetString(flagKeysDir)
		keyringBackend, _ := cmd.Flags().GetString(flagKeyringBackend)
		keyringDir, _ := cmd.Flags().GetString(flagKeyringDir)

		config.Keys.KeysFile = keysFile
		config.Keys.KeysDir = keysDir
		if keyringBackend != "" {
			if keyringDir == "" {
				keyringDir = os.ExpandEnv("$HOME/.pocket")
			}
			config.Keys.Keyring = &miner.KeyringConfig{
				Backend: keyringBackend,
				Dir:     keyringDir,
			}
		}

		// Validate key sources
		if !config.HasKeySource() {
			return nil, fmt.Errorf("at least one key source must be specified: --config, --keys-file, --keys-dir, or --keyring-backend")
		}
	}

	// Apply flag overrides (flags take precedence over config file)
	applyFlagOverrides(cmd, config)

	// Generate consumer name from hostname if not set
	if config.Redis.ConsumerName == "" {
		hostname, _ := os.Hostname()
		config.Redis.ConsumerName = fmt.Sprintf("miner-%s-%d", hostname, os.Getpid())
	}

	return config, nil
}

// applyFlagOverrides applies command-line flag overrides to the config.
func applyFlagOverrides(cmd *cobra.Command, config *miner.Config) {
	if cmd.Flags().Changed(flagRedisURL) {
		redisURL, _ := cmd.Flags().GetString(flagRedisURL)
		config.Redis.URL = redisURL
	}
	if cmd.Flags().Changed(flagConsumerGroup) {
		consumerGroup, _ := cmd.Flags().GetString(flagConsumerGroup)
		config.Redis.ConsumerGroup = consumerGroup
	}
	if cmd.Flags().Changed(flagConsumerName) {
		consumerName, _ := cmd.Flags().GetString(flagConsumerName)
		config.Redis.ConsumerName = consumerName
	}
	if cmd.Flags().Changed(flagStreamPrefix) {
		streamPrefix, _ := cmd.Flags().GetString(flagStreamPrefix)
		config.Redis.StreamPrefix = streamPrefix
	}
	if cmd.Flags().Changed(flagHotReload) {
		hotReload, _ := cmd.Flags().GetBool(flagHotReload)
		config.HotReloadEnabled = hotReload
	}
	if cmd.Flags().Changed(flagSessionTTL) {
		sessionTTL, _ := cmd.Flags().GetDuration(flagSessionTTL)
		config.SessionTTL = sessionTTL
	}
	if cmd.Flags().Changed(flagWALMaxLen) {
		walMaxLen, _ := cmd.Flags().GetInt64(flagWALMaxLen)
		config.WALMaxLen = walMaxLen
	}
}

// createKeyProviders creates key providers based on the config.
func createKeyProviders(logger polylog.Logger, config *miner.Config) ([]keys.KeyProvider, error) {
	var providers []keys.KeyProvider

	if config.Keys.KeysFile != "" {
		provider, err := keys.NewSupplierKeysFileProvider(logger, config.Keys.KeysFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create supplier keys file provider: %w", err)
		}
		providers = append(providers, provider)
		logger.Info().Str("file", config.Keys.KeysFile).Msg("added supplier keys file provider")
	}

	if config.Keys.KeysDir != "" {
		provider, err := keys.NewFileKeyProvider(logger, config.Keys.KeysDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create file key provider: %w", err)
		}
		providers = append(providers, provider)
		logger.Info().Str("dir", config.Keys.KeysDir).Msg("added file key provider")
	}

	if config.Keys.Keyring != nil && config.Keys.Keyring.Backend != "" {
		keyringDir := config.Keys.Keyring.Dir
		if keyringDir == "" {
			keyringDir = os.ExpandEnv("$HOME/.pocket")
		}
		provider, err := keys.NewKeyringProvider(logger, keys.KeyringProviderConfig{
			Backend:  config.Keys.Keyring.Backend,
			Dir:      keyringDir,
			AppName:  config.Keys.Keyring.AppName,
			KeyNames: config.Keys.Keyring.KeyNames,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create keyring provider: %w", err)
		}
		providers = append(providers, provider)
		logger.Info().
			Str("backend", config.Keys.Keyring.Backend).
			Str("dir", keyringDir).
			Msg("added keyring provider")
	}

	return providers, nil
}
