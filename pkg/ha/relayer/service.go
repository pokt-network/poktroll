package relayer

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/ha/cache"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	redistransport "github.com/pokt-network/poktroll/pkg/ha/transport/redis"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// Service is the main HA Relayer service that orchestrates all components.
// It manages the proxy server, health checker, validator, and Redis transport.
type Service struct {
	logger polylog.Logger
	config *Config

	// Core components
	proxyServer   *ProxyServer
	healthChecker *HealthChecker
	validator     RelayValidator
	publisher     transport.MinedRelayPublisher

	// Redis client
	redisClient redis.UniversalClient

	// Caches (created from Redis client)
	sharedParamCache cache.SharedParamCache
	sessionCache     cache.SessionCache

	// Block subscriber for height updates
	blockSubscriber cache.BlockHeightSubscriber

	// Ring client for signature verification
	ringClient crypto.RingClient

	// Lifecycle
	mu       sync.Mutex
	started  bool
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// ServiceDependencies contains external dependencies required by the Service.
type ServiceDependencies struct {
	// Logger is the logger instance.
	Logger polylog.Logger

	// RedisClient is the Redis client for pub/sub and caching.
	RedisClient redis.UniversalClient

	// RingClient is used for ring signature verification.
	RingClient crypto.RingClient

	// SessionClient is used for session queries.
	SessionClient client.SessionQueryClient

	// SharedClient is used for shared param queries.
	SharedClient client.SharedQueryClient

	// BlockClient is used for block height queries.
	BlockClient client.BlockClient
}

// NewService creates a new HA Relayer service.
func NewService(config *Config, deps ServiceDependencies) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := logging.ForComponent(deps.Logger, logging.ComponentService)

	// Create cache config
	cacheConfig := cache.CacheConfig{
		RedisURL:               config.Redis.URL,
		CachePrefix:            config.Redis.StreamPrefix + ":cache",
		PubSubPrefix:           config.Redis.StreamPrefix + ":events",
		ExtraGracePeriodBlocks: config.GracePeriodExtraBlocks,
	}

	// Create shared param cache
	sharedParamCache := cache.NewRedisSharedParamCache(
		logger,
		deps.RedisClient,
		deps.SharedClient,
		deps.BlockClient,
		cacheConfig,
	)

	// Create session cache
	sessionCache := cache.NewRedisSessionCache(
		logger,
		deps.RedisClient,
		deps.SessionClient,
		deps.SharedClient,
		deps.BlockClient,
		cacheConfig,
	)

	// Create block subscriber
	blockSubscriber := cache.NewRedisBlockSubscriber(
		logger,
		deps.RedisClient,
		deps.BlockClient,
		cacheConfig,
	)

	// Create health checker
	healthChecker := NewHealthChecker(logger)

	// Register backends for health checking (per RPC type)
	for serviceID, svcConfig := range config.Services {
		for rpcType, backend := range svcConfig.Backends {
			backendID := fmt.Sprintf("%s:%s", serviceID, rpcType)
			healthChecker.RegisterBackend(backendID, backend.URL, backend.HealthCheck)
		}
	}

	// Create Redis publisher
	publisherConfig := transport.PublisherConfig{
		StreamPrefix: config.Redis.StreamPrefix,
		MaxLen:       config.Redis.MaxStreamLen,
		ApproxMaxLen: true, // Use approximate trimming for better performance
	}
	if publisherConfig.MaxLen == 0 {
		publisherConfig.MaxLen = 100000 // Default max stream length
	}

	publisher := redistransport.NewStreamsPublisher(
		logger,
		deps.RedisClient,
		publisherConfig,
	)

	// Create proxy server
	proxyServer, err := NewProxyServer(logger, config, healthChecker, publisher)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy server: %w", err)
	}

	// Create validator (if ring client is provided)
	var validator RelayValidator
	if deps.RingClient != nil {
		validatorConfig := &ValidatorConfig{
			GracePeriodExtraBlocks: config.GracePeriodExtraBlocks,
		}
		validator = NewRelayValidator(
			logger,
			validatorConfig,
			deps.RingClient,
			sessionCache,
			sharedParamCache,
		)
		proxyServer.SetValidator(validator)
	}

	return &Service{
		logger:           logger,
		config:           config,
		proxyServer:      proxyServer,
		healthChecker:    healthChecker,
		validator:        validator,
		publisher:        publisher,
		redisClient:      deps.RedisClient,
		sharedParamCache: sharedParamCache,
		sessionCache:     sessionCache,
		blockSubscriber:  blockSubscriber,
		ringClient:       deps.RingClient,
	}, nil
}

// Start starts the HA Relayer service.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("service is closed")
	}
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("service already started")
	}

	s.started = true
	ctx, s.cancelFn = context.WithCancel(ctx)
	s.mu.Unlock()

	s.logger.Info().Msg("starting HA Relayer service")

	// Start caches
	if err := s.sharedParamCache.Start(ctx); err != nil {
		return fmt.Errorf("failed to start shared param cache: %w", err)
	}

	if err := s.sessionCache.Start(ctx); err != nil {
		return fmt.Errorf("failed to start session cache: %w", err)
	}

	// Start block subscriber
	if err := s.blockSubscriber.Start(ctx); err != nil {
		return fmt.Errorf("failed to start block subscriber: %w", err)
	}

	// Subscribe to block height updates
	s.wg.Add(1)
	go s.subscribeToBlocks(ctx)

	// Start health checker
	if err := s.healthChecker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	// Start proxy server
	if err := s.proxyServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	s.logger.Info().
		Str(logging.FieldListenAddr, s.config.ListenAddr).
		Int("num_services", len(s.config.Services)).
		Msg("HA Relayer service started")

	return nil
}

// subscribeToBlocks subscribes to block height updates and updates the proxy.
func (s *Service) subscribeToBlocks(ctx context.Context) {
	defer s.wg.Done()

	blockCh := s.blockSubscriber.Subscribe(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-blockCh:
			if !ok {
				return
			}
			s.proxyServer.SetBlockHeight(event.Height)
			s.logger.Debug().
				Int64("height", event.Height).
				Msg("updated block height")
		}
	}
}

// Stop gracefully stops the HA Relayer service.
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.logger.Info().Msg("stopping HA Relayer service")

	// Cancel context to signal shutdown
	if s.cancelFn != nil {
		s.cancelFn()
	}

	var errs []error

	// Stop components in reverse order
	if err := s.proxyServer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close proxy server: %w", err))
	}

	if err := s.healthChecker.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close health checker: %w", err))
	}

	if err := s.publisher.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close publisher: %w", err))
	}

	if err := s.blockSubscriber.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close block subscriber: %w", err))
	}

	if err := s.sessionCache.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close session cache: %w", err))
	}

	if err := s.sharedParamCache.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close shared param cache: %w", err))
	}

	// Wait for goroutines
	s.wg.Wait()

	s.logger.Info().Msg("HA Relayer service stopped")

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

// GetProxyServer returns the proxy server for testing/monitoring purposes.
func (s *Service) GetProxyServer() *ProxyServer {
	return s.proxyServer
}

// GetHealthChecker returns the health checker for testing/monitoring purposes.
func (s *Service) GetHealthChecker() *HealthChecker {
	return s.healthChecker
}

// GetPublisher returns the Redis publisher for testing purposes.
func (s *Service) GetPublisher() transport.MinedRelayPublisher {
	return s.publisher
}
