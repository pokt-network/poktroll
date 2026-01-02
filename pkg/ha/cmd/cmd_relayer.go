package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/ha/cache"
	"github.com/pokt-network/poktroll/pkg/ha/keys"
	"github.com/pokt-network/poktroll/pkg/ha/observability"
	"github.com/pokt-network/poktroll/pkg/ha/query"
	"github.com/pokt-network/poktroll/pkg/ha/relayer"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	redistransport "github.com/pokt-network/poktroll/pkg/ha/transport/redis"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

const (
	flagRelayerConfig = "config"
	flagRedisURL      = "redis-url"
)

// startRelayerCmd returns the command for starting the HA Relayer component.
func startRelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayer",
		Short: "Start the HA Relayer (HTTP/WebSocket proxy)",
		Long: `Start the High-Availability Relayer component.

The HA Relayer handles incoming relay requests and forwards them to backend services.
It is stateless and can be scaled horizontally behind a load balancer.

Features:
- HTTP and WebSocket relay proxying
- Request validation and signing
- Health checking for backends
- Prometheus metrics at /metrics

Example:
  pocketd relayminer ha relayer --config /path/to/ha-relayer.yaml --redis-url redis://localhost:6379
`,
		RunE: runHARelayer,
	}

	cmd.Flags().String(flagRelayerConfig, "", "Path to HA relayer config file (required)")
	cmd.Flags().String(flagRedisURL, "redis://localhost:6379", "Redis connection URL")

	_ = cmd.MarkFlagRequired(flagRelayerConfig)

	return cmd
}

func runHARelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Set up logger
	logger := polyzero.NewLogger()

	// Load config
	configPath, _ := cmd.Flags().GetString(flagRelayerConfig)
	config, err := relayer.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Start observability server (metrics)
	if config.Metrics.Enabled {
		obsServer := observability.NewServer(logger, observability.ServerConfig{
			MetricsEnabled: config.Metrics.Enabled,
			MetricsAddr:    config.Metrics.Addr,
			PprofEnabled:   false,
			Registry:       observability.RelayerRegistry,
		})
		if err := obsServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start observability server: %w", err)
		}
		defer obsServer.Stop()
		logger.Info().Str("addr", config.Metrics.Addr).Msg("observability server started")
	}

	// Use Redis URL from config, allow flag override
	redisURL := config.Redis.URL
	if cmd.Flags().Changed(flagRedisURL) {
		redisURL, _ = cmd.Flags().GetString(flagRedisURL)
	}
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// Test Redis connection
	if pingErr := redisClient.Ping(ctx).Err(); pingErr != nil {
		return fmt.Errorf("failed to connect to Redis: %w", pingErr)
	}
	logger.Info().Str("redis_url", redisURL).Msg("connected to Redis")

	// Create supplier cache for checking supplier staking state
	supplierCache := cache.NewSupplierCache(
		logger,
		redisClient,
		cache.SupplierCacheConfig{
			KeyPrefix: "ha:supplier",
			FailOpen:  true, // Prioritize serving traffic over strict validation
		},
	)
	logger.Info().Msg("supplier cache initialized")

	// Create query clients for fetching on-chain data (service compute units, etc.)
	// Determine TLS from URL - if port 443 or grpcs:// prefix, use TLS
	grpcURL := config.PocketNode.QueryNodeGRPCUrl
	useTLS := strings.HasPrefix(grpcURL, "grpcs://") ||
		strings.HasPrefix(grpcURL, "https://") ||
		strings.HasSuffix(grpcURL, ":443")
	// Strip scheme for gRPC endpoint
	grpcEndpoint := grpcURL
	grpcEndpoint = strings.TrimPrefix(grpcEndpoint, "grpcs://")
	grpcEndpoint = strings.TrimPrefix(grpcEndpoint, "grpc://")
	grpcEndpoint = strings.TrimPrefix(grpcEndpoint, "https://")
	grpcEndpoint = strings.TrimPrefix(grpcEndpoint, "http://")

	queryClients, err := query.NewQueryClients(
		logger,
		query.QueryClientConfig{
			GRPCEndpoint: grpcEndpoint,
			QueryTimeout: 30 * time.Second,
			UseTLS:       useTLS,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create query clients: %w", err)
	}
	defer queryClients.Close()
	logger.Info().Str("grpc_endpoint", grpcEndpoint).Bool("tls", useTLS).Msg("query clients initialized")

	// Create publisher for mined relays
	publisher := redistransport.NewStreamsPublisher(
		logger,
		redisClient,
		transport.PublisherConfig{
			StreamPrefix: config.Redis.StreamPrefix,
			MaxLen:       config.Redis.MaxStreamLen,
			ApproxMaxLen: true,
		},
	)

	// Create health checker
	healthChecker := relayer.NewHealthChecker(logger)

	// Create proxy server
	proxy, err := relayer.NewProxyServer(
		logger,
		config,
		healthChecker,
		publisher,
	)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	// Load keys and create response signer
	// Support multiple key sources: keys_file, keys_dir, keyring
	var keyProviders []keys.KeyProvider

	// Try keys_file first (preferred for HA setup - same as miner)
	if config.Keys.KeysFile != "" {
		provider, keyErr := keys.NewSupplierKeysFileProvider(logger, config.Keys.KeysFile)
		if keyErr != nil {
			return fmt.Errorf("failed to create supplier keys file provider: %w", keyErr)
		}
		keyProviders = append(keyProviders, provider)
		logger.Info().Str("file", config.Keys.KeysFile).Msg("added supplier keys file provider")
	}

	// Try keyring as additional source (can combine both)
	if config.Keys.Keyring != nil && config.Keys.Keyring.Backend != "" {
		provider, keyErr := keys.NewKeyringProvider(logger, keys.KeyringProviderConfig{
			Backend:  config.Keys.Keyring.Backend,
			Dir:      config.Keys.Keyring.Dir,
			AppName:  config.Keys.Keyring.AppName,
			KeyNames: config.Keys.Keyring.KeyNames,
		})
		if keyErr != nil {
			return fmt.Errorf("failed to create keyring provider: %w", keyErr)
		}
		keyProviders = append(keyProviders, provider)
		logger.Info().Str("backend", config.Keys.Keyring.Backend).Msg("added keyring provider")
	}

	if len(keyProviders) == 0 {
		logger.Warn().Msg("no key providers configured - response signing will be disabled (relays will fail)")
	} else {
		// Load keys from all providers (both return map[string]cryptotypes.PrivKey)
		loadedKeys := make(map[string]cryptotypes.PrivKey)
		for _, provider := range keyProviders {
			providerKeys, loadErr := provider.LoadKeys(ctx)
			if loadErr != nil {
				logger.Warn().Err(loadErr).Str("provider", provider.Name()).Msg("failed to load keys from provider")
				continue
			}
			for addr, key := range providerKeys {
				loadedKeys[addr] = key
			}
			logger.Info().Str("provider", provider.Name()).Int("keys", len(providerKeys)).Msg("loaded keys from provider")
		}

		// Close providers
		for _, provider := range keyProviders {
			provider.Close()
		}

		if len(loadedKeys) == 0 {
			logger.Warn().Msg("no keys found - response signing will be disabled")
		} else {
			responseSigner, signerErr := relayer.NewResponseSigner(logger, loadedKeys)
			if signerErr != nil {
				return fmt.Errorf("failed to create response signer: %w", signerErr)
			}
			proxy.SetResponseSigner(responseSigner)
			logger.Info().
				Int("num_keys", len(responseSigner.GetOperatorAddresses())).
				Str("operator_addresses", strings.Join(responseSigner.GetOperatorAddresses(), ", ")).
				Msg("response signer initialized")

			// Create RelayProcessor for proper relay mining with session metadata
			signerAdapter := relayer.NewResponseSignerAdapter(responseSigner)
			relayProcessor := relayer.NewRelayProcessor(
				logger,
				publisher,
				signerAdapter,
				nil, // ringClient - not needed for HA relayer as we don't verify requests
			)
			// Wire up the service compute units provider using on-chain service data
			computeUnitsProvider := relayer.NewCachedServiceComputeUnitsProvider(logger, queryClients.Service())
			// Preload compute units for configured services
			serviceIDs := make([]string, 0, len(config.Services))
			for serviceID := range config.Services {
				serviceIDs = append(serviceIDs, serviceID)
			}
			computeUnitsProvider.PreloadServiceComputeUnits(ctx, serviceIDs)
			relayProcessor.SetServiceComputeUnitsProvider(computeUnitsProvider)
			proxy.SetRelayProcessor(relayProcessor)
			logger.Info().Msg("relay processor initialized")

			// Initialize gRPC handler for gRPC and gRPC-Web requests
			proxy.InitGRPCHandler()
		}
	}

	// Set supplier cache for checking supplier state before accepting relays
	proxy.SetSupplierCache(supplierCache)

	// Register backends for health checking (per RPC type)
	for serviceID, svc := range config.Services {
		for rpcType, backend := range svc.Backends {
			if backend.HealthCheck != nil && backend.HealthCheck.Enabled {
				backendID := fmt.Sprintf("%s:%s", serviceID, rpcType)
				healthChecker.RegisterBackend(backendID, backend.URL, backend.HealthCheck)
			}
		}
	}

	// Start components
	if err := healthChecker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	if err := proxy.Start(ctx); err != nil {
		return fmt.Errorf("failed to start proxy: %w", err)
	}

	logger.Info().
		Str("listen_addr", config.ListenAddr).
		Int("num_services", len(config.Services)).
		Msg("HA Relayer started")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info().Msg("shutdown signal received, stopping HA Relayer...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	proxy.Close()
	healthChecker.Close()

	_ = shutdownCtx // Used for graceful shutdown timing

	logger.Info().Msg("HA Relayer stopped")
	return nil
}
