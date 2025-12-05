package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ SessionCache = (*RedisSessionCache)(nil)

// RedisSessionCache implements SessionCache using Redis as L2 cache.
type RedisSessionCache struct {
	logger        polylog.Logger
	redisClient   redis.UniversalClient
	sessionClient client.SessionQueryClient
	sharedClient  client.SharedQueryClient
	blockClient   client.BlockClient
	config        CacheConfig

	// L1 local cache for sessions
	sessionCache sync.Map // map[string]*sessiontypes.Session

	// L1 local cache for session rewardability (fast path)
	rewardableCache sync.Map // map[string]bool (true = rewardable, false = non-rewardable)

	// Cache keys helper
	keys CacheKeys

	// Lifecycle
	mu       sync.RWMutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewRedisSessionCache creates a new SessionCache backed by Redis.
func NewRedisSessionCache(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	sessionClient client.SessionQueryClient,
	sharedClient client.SharedQueryClient,
	blockClient client.BlockClient,
	config CacheConfig,
) *RedisSessionCache {
	if config.CachePrefix == "" {
		config.CachePrefix = "ha:cache"
	}
	if config.PubSubPrefix == "" {
		config.PubSubPrefix = "ha:events"
	}
	if config.BlockTimeSeconds == 0 {
		config.BlockTimeSeconds = 6
	}
	if config.ExtraGracePeriodBlocks == 0 {
		config.ExtraGracePeriodBlocks = 2
	}

	return &RedisSessionCache{
		logger:        logging.ForComponent(logger, logging.ComponentSessionCache),
		redisClient:   redisClient,
		sessionClient: sessionClient,
		sharedClient:  sharedClient,
		blockClient:   blockClient,
		config:        config,
		keys:          CacheKeys{Prefix: config.CachePrefix},
	}
}

// Start begins the cache's background processes.
func (c *RedisSessionCache) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("cache is closed")
	}

	ctx, c.cancelFn = context.WithCancel(ctx)
	c.mu.Unlock()

	// Subscribe to session rewardability updates
	c.wg.Add(1)
	go c.subscribeToRewardabilityUpdates(ctx)

	c.logger.Info().Msg("session cache started")
	return nil
}

// subscribeToRewardabilityUpdates listens for session rewardability changes from other instances.
func (c *RedisSessionCache) subscribeToRewardabilityUpdates(ctx context.Context) {
	defer c.wg.Done()

	channel := c.config.PubSubPrefix + ":session:rewardable"
	pubsub := c.redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-pubsub.Channel():
			var update SessionRewardableUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
				c.logger.Warn().Err(err).Str("payload", msg.Payload).Msg("invalid rewardability update")
				continue
			}

			// Update L1 cache
			c.rewardableCache.Store(update.SessionID, update.IsRewardable)

			if !update.IsRewardable {
				sessionMarkedNonRewardable.WithLabelValues(update.Reason).Inc()
			}
		}
	}
}

// SessionRewardableUpdate is the pub/sub message for session rewardability changes.
type SessionRewardableUpdate struct {
	SessionID    string `json:"session_id"`
	IsRewardable bool   `json:"is_rewardable"`
	Reason       string `json:"reason,omitempty"`
}

// GetSession returns the session for the given application, service, and block height.
func (c *RedisSessionCache) GetSession(ctx context.Context, appAddress, serviceId string, height int64) (*sessiontypes.Session, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	key := c.keys.Session(appAddress, serviceId, height)

	// L1: Check local cache
	if cached, ok := c.sessionCache.Load(key); ok {
		cacheHits.WithLabelValues("session", "l1").Inc()
		return cached.(*sessiontypes.Session), nil
	}
	cacheMisses.WithLabelValues("session", "l1").Inc()

	// L2: Check Redis cache
	data, err := c.redisClient.Get(ctx, key).Bytes()
	if err == nil {
		session := &sessiontypes.Session{}
		if unmarshalErr := json.Unmarshal(data, session); unmarshalErr != nil {
			c.logger.Warn().Err(unmarshalErr).Msg("failed to unmarshal cached session")
		} else {
			cacheHits.WithLabelValues("session", "l2").Inc()
			c.sessionCache.Store(key, session)
			return session, nil
		}
	}
	if err != nil && err != redis.Nil {
		c.logger.Warn().Err(err).Msg("error fetching session from Redis")
	}
	cacheMisses.WithLabelValues("session", "l2").Inc()

	// L3: Query chain
	session, err := c.sessionClient.GetSession(ctx, appAddress, serviceId, height)
	if err != nil {
		chainQueryErrors.WithLabelValues("session").Inc()
		return nil, fmt.Errorf("failed to query session: %w", err)
	}
	chainQueries.WithLabelValues("session").Inc()

	// Calculate TTL based on session end height
	ttl := c.calculateSessionTTL(ctx, session.Header.SessionEndBlockHeight)

	// Cache in Redis
	data, err = json.Marshal(session)
	if err == nil {
		if err := c.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
			c.logger.Warn().Err(err).Msg("failed to cache session in Redis")
		}
	}

	// Cache in L1
	c.sessionCache.Store(key, session)

	return session, nil
}

// calculateSessionTTL calculates how long to cache a session.
func (c *RedisSessionCache) calculateSessionTTL(ctx context.Context, sessionEndHeight int64) time.Duration {
	currentBlock := c.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()

	// Get shared params for grace period
	sharedParams, err := c.sharedClient.GetParams(ctx)
	if err != nil {
		// Fall back to default TTL on error
		return time.Duration(c.config.BlockTimeSeconds*10) * time.Second
	}

	// Calculate when session is no longer valid (end + grace + extra grace)
	onChainGrace := int64(sharedParams.GetGracePeriodEndOffsetBlocks())
	extraGrace := c.config.ExtraGracePeriodBlocks
	sessionValidUntil := sessionEndHeight + onChainGrace + extraGrace

	// If session is already past validity, short TTL
	if currentHeight >= sessionValidUntil {
		return time.Duration(c.config.BlockTimeSeconds) * time.Second
	}

	// TTL = blocks remaining * block time + buffer
	blocksRemaining := sessionValidUntil - currentHeight
	return time.Duration(blocksRemaining*c.config.BlockTimeSeconds+10) * time.Second
}

// GetSessionValidation returns cached validation result for a session.
func (c *RedisSessionCache) GetSessionValidation(ctx context.Context, appAddress, serviceId string, height int64) (*SessionValidationResult, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	key := c.keys.SessionValidation(appAddress, serviceId, height)

	data, err := c.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No cached result
		}
		return nil, fmt.Errorf("failed to get validation result: %w", err)
	}

	var result SessionValidationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation result: %w", err)
	}

	cacheHits.WithLabelValues("session_validation", "l2").Inc()
	return &result, nil
}

// SetSessionValidation caches a session validation result.
func (c *RedisSessionCache) SetSessionValidation(ctx context.Context, result *SessionValidationResult) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	key := c.keys.SessionValidation(result.AppAddress, result.ServiceId, result.BlockHeight)

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal validation result: %w", err)
	}

	// Short TTL for validation results (1 block)
	ttl := time.Duration(c.config.BlockTimeSeconds) * time.Second

	if err := c.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache validation result: %w", err)
	}

	return nil
}

// IsSessionRewardable checks if a session is still eligible for rewards.
func (c *RedisSessionCache) IsSessionRewardable(ctx context.Context, sessionId string) bool {
	// L1: Check local cache first
	if cached, ok := c.rewardableCache.Load(sessionId); ok {
		isRewardable := cached.(bool)
		if isRewardable {
			sessionRewardableChecks.WithLabelValues("rewardable").Inc()
		} else {
			sessionRewardableChecks.WithLabelValues("non_rewardable").Inc()
		}
		return isRewardable
	}

	// L2: Check Redis
	key := c.keys.SessionRewardable(sessionId)
	val, err := c.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Not found = assume rewardable (default state)
		sessionRewardableChecks.WithLabelValues("rewardable").Inc()
		return true
	}
	if err != nil {
		// Error - assume rewardable to avoid false rejections
		c.logger.Warn().Err(err).Str(logging.FieldSessionID, sessionId).Msg("error checking rewardability")
		sessionRewardableChecks.WithLabelValues("rewardable").Inc()
		return true
	}

	// Found - check value
	isRewardable := val != "false"
	c.rewardableCache.Store(sessionId, isRewardable) // Update L1

	if isRewardable {
		sessionRewardableChecks.WithLabelValues("rewardable").Inc()
	} else {
		sessionRewardableChecks.WithLabelValues("non_rewardable").Inc()
	}

	return isRewardable
}

// MarkSessionNonRewardable marks a session as no longer eligible for rewards.
func (c *RedisSessionCache) MarkSessionNonRewardable(ctx context.Context, sessionId string, reason string) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	key := c.keys.SessionRewardable(sessionId)

	// Get session TTL from shared params
	sharedParams, err := c.sharedClient.GetParams(ctx)
	if err != nil {
		// Use default TTL on error
		sharedParams = &sharedtypes.Params{
			NumBlocksPerSession:          10,
			GracePeriodEndOffsetBlocks:   1,
			ClaimWindowOpenOffsetBlocks:  1,
			ClaimWindowCloseOffsetBlocks: 4,
		}
	}

	// Calculate TTL (session duration + all windows)
	numBlocks := int64(sharedParams.GetNumBlocksPerSession())
	graceBlocks := int64(sharedParams.GetGracePeriodEndOffsetBlocks())
	claimOpenBlocks := int64(sharedParams.GetClaimWindowOpenOffsetBlocks())
	claimCloseBlocks := int64(sharedParams.GetClaimWindowCloseOffsetBlocks())
	totalBlocks := numBlocks + graceBlocks + claimOpenBlocks + claimCloseBlocks + c.config.ExtraGracePeriodBlocks
	ttl := time.Duration(totalBlocks*c.config.BlockTimeSeconds) * time.Second

	// Set in Redis
	if err := c.redisClient.Set(ctx, key, "false", ttl).Err(); err != nil {
		return fmt.Errorf("failed to mark session non-rewardable: %w", err)
	}

	// Update L1
	c.rewardableCache.Store(sessionId, false)

	// Publish to notify other instances
	channel := c.config.PubSubPrefix + ":session:rewardable"
	update := SessionRewardableUpdate{
		SessionID:    sessionId,
		IsRewardable: false,
		Reason:       reason,
	}
	data, _ := json.Marshal(update)
	if err := c.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		c.logger.Warn().Err(err).Msg("failed to publish rewardability update")
	}

	sessionMarkedNonRewardable.WithLabelValues(reason).Inc()
	c.logger.Debug().
		Str(logging.FieldSessionID, sessionId).
		Str(logging.FieldReason, reason).
		Msg("marked session as non-rewardable")

	return nil
}

// Close gracefully shuts down the cache.
func (c *RedisSessionCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.cancelFn != nil {
		c.cancelFn()
	}

	c.wg.Wait()

	c.logger.Info().Msg("session cache closed")
	return nil
}
