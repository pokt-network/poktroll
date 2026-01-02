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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ SharedParamCache = (*RedisSharedParamCache)(nil)

// RedisSharedParamCache implements SharedParamCache using Redis as L2 cache.
type RedisSharedParamCache struct {
	logger       polylog.Logger
	redisClient  redis.UniversalClient
	sharedClient client.SharedQueryClient
	blockClient  client.BlockClient
	config       CacheConfig

	// L1 local cache
	localCache sync.Map // map[string]*sharedtypes.Params

	// Cache keys helper
	keys CacheKeys

	// Lifecycle
	mu       sync.RWMutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewRedisSharedParamCache creates a new SharedParamCache backed by Redis.
func NewRedisSharedParamCache(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	sharedClient client.SharedQueryClient,
	blockClient client.BlockClient,
	config CacheConfig,
) *RedisSharedParamCache {
	if config.CachePrefix == "" {
		config.CachePrefix = "ha:cache"
	}
	if config.TTLBlocks == 0 {
		config.TTLBlocks = 1
	}
	if config.BlockTimeSeconds == 0 {
		config.BlockTimeSeconds = 6
	}
	if config.LockTimeout == 0 {
		config.LockTimeout = 5 * time.Second
	}

	return &RedisSharedParamCache{
		logger:       logging.ForComponent(logger, logging.ComponentSharedParamCache),
		redisClient:  redisClient,
		sharedClient: sharedClient,
		blockClient:  blockClient,
		config:       config,
		keys:         CacheKeys{Prefix: config.CachePrefix},
	}
}

// Start begins the cache's background processes.
func (c *RedisSharedParamCache) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("cache is closed")
	}

	ctx, c.cancelFn = context.WithCancel(ctx)
	c.mu.Unlock()

	// Subscribe to cache invalidation events
	c.wg.Add(1)
	go c.subscribeToInvalidations(ctx)

	c.logger.Info().Msg("shared param cache started")
	return nil
}

// subscribeToInvalidations listens for cache invalidation events from other instances.
func (c *RedisSharedParamCache) subscribeToInvalidations(ctx context.Context) {
	defer c.wg.Done()

	channel := c.config.PubSubPrefix + ":invalidate:params"
	pubsub := c.redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-pubsub.Channel():
			// Parse the height from the message
			var height int64
			if _, err := fmt.Sscanf(msg.Payload, "%d", &height); err != nil {
				c.logger.Warn().Err(err).Str("payload", msg.Payload).Msg("invalid invalidation message")
				continue
			}

			// Clear local cache for this height
			key := c.keys.SharedParams(height)
			c.localCache.Delete(key)
			cacheInvalidations.WithLabelValues("shared_params", "pubsub").Inc()
		}
	}
}

// GetSharedParams returns the shared module parameters for the given block height.
func (c *RedisSharedParamCache) GetSharedParams(ctx context.Context, height int64) (*sharedtypes.Params, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	key := c.keys.SharedParams(height)

	// L1: Check local cache
	if cached, ok := c.localCache.Load(key); ok {
		cacheHits.WithLabelValues("shared_params", "l1").Inc()
		return cached.(*sharedtypes.Params), nil
	}
	cacheMisses.WithLabelValues("shared_params", "l1").Inc()

	// L2: Check Redis cache
	data, err := c.redisClient.Get(ctx, key).Bytes()
	if err == nil {
		params := &sharedtypes.Params{}
		if unmarshalErr := json.Unmarshal(data, params); unmarshalErr != nil {
			c.logger.Warn().Err(unmarshalErr).Msg("failed to unmarshal cached params")
		} else {
			cacheHits.WithLabelValues("shared_params", "l2").Inc()
			// Store in L1
			c.localCache.Store(key, params)
			return params, nil
		}
	}
	if err != nil && err != redis.Nil {
		c.logger.Warn().Err(err).Msg("error fetching from Redis cache")
	}
	cacheMisses.WithLabelValues("shared_params", "l2").Inc()

	// L3: Query chain with distributed lock
	return c.queryAndCacheParams(ctx, height, key)
}

// queryAndCacheParams queries the chain and caches the result.
// Uses distributed locking to prevent thundering herd.
func (c *RedisSharedParamCache) queryAndCacheParams(ctx context.Context, height int64, key string) (*sharedtypes.Params, error) {
	lockKey := c.keys.SharedParamsLock(height)

	// Try to acquire lock
	locked, err := c.redisClient.SetNX(ctx, lockKey, "1", c.config.LockTimeout).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if locked {
		// We got the lock - query chain
		defer c.redisClient.Del(ctx, lockKey)

		params, queryErr := c.sharedClient.GetParams(ctx)
		if queryErr != nil {
			chainQueryErrors.WithLabelValues("shared_params").Inc()
			return nil, fmt.Errorf("failed to query chain: %w", queryErr)
		}
		chainQueries.WithLabelValues("shared_params").Inc()

		// Cache in Redis
		data, marshalErr := json.Marshal(params)
		if marshalErr == nil {
			ttl := c.config.BlocksToTTL(c.config.TTLBlocks)
			if cacheErr := c.redisClient.Set(ctx, key, data, ttl).Err(); cacheErr != nil {
				c.logger.Warn().Err(cacheErr).Msg("failed to cache params in Redis")
			}
		}

		// Cache in L1
		c.localCache.Store(key, params)

		return params, nil
	}

	// Another instance is populating - wait and retry from Redis
	time.Sleep(100 * time.Millisecond)

	retryData, retryErr := c.redisClient.Get(ctx, key).Bytes()
	if retryErr == nil {
		params := &sharedtypes.Params{}
		if unmarshalErr := json.Unmarshal(retryData, params); unmarshalErr == nil {
			cacheHits.WithLabelValues("shared_params", "l2_retry").Inc()
			c.localCache.Store(key, params)
			return params, nil
		}
	}

	// Still not available - query chain directly
	params, fallbackErr := c.sharedClient.GetParams(ctx)
	if fallbackErr != nil {
		chainQueryErrors.WithLabelValues("shared_params").Inc()
		return nil, fmt.Errorf("failed to query chain: %w", fallbackErr)
	}
	chainQueries.WithLabelValues("shared_params").Inc()

	return params, nil
}

// GetLatestSharedParams returns the shared module parameters for the latest block.
func (c *RedisSharedParamCache) GetLatestSharedParams(ctx context.Context) (*sharedtypes.Params, error) {
	latestBlock := c.blockClient.LastBlock(ctx)
	return c.GetSharedParams(ctx, latestBlock.Height())
}

// InvalidateSharedParams invalidates the cached shared params for a specific height.
func (c *RedisSharedParamCache) InvalidateSharedParams(ctx context.Context, height int64) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("cache is closed")
	}
	c.mu.RUnlock()

	key := c.keys.SharedParams(height)

	// Clear L1
	c.localCache.Delete(key)

	// Clear L2
	if err := c.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	// Notify other instances
	channel := c.config.PubSubPrefix + ":invalidate:params"
	if err := c.redisClient.Publish(ctx, channel, fmt.Sprintf("%d", height)).Err(); err != nil {
		c.logger.Warn().Err(err).Msg("failed to publish invalidation")
	}

	cacheInvalidations.WithLabelValues("shared_params", "manual").Inc()
	return nil
}

// Close gracefully shuts down the cache.
func (c *RedisSharedParamCache) Close() error {
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

	c.logger.Info().Msg("shared param cache closed")
	return nil
}
