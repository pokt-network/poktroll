package miner

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// Deduplicator ensures relays are processed only once across all Miner instances.
// It uses Redis for distributed coordination with local caching for performance.
type Deduplicator interface {
	// IsDuplicate checks if a relay has already been processed.
	// Returns true if this relay hash has been seen before.
	IsDuplicate(ctx context.Context, relayHash []byte, sessionID string) (bool, error)

	// MarkProcessed marks a relay hash as processed.
	// This should be called after successful processing.
	MarkProcessed(ctx context.Context, relayHash []byte, sessionID string) error

	// MarkProcessedBatch marks multiple relay hashes as processed.
	MarkProcessedBatch(ctx context.Context, relayHashes [][]byte, sessionID string) error

	// CleanupSession removes all deduplication entries for a session.
	// Call this after a session's claim window has closed.
	CleanupSession(ctx context.Context, sessionID string) error

	// Start begins the deduplicator's background processes.
	Start(ctx context.Context) error

	// Close gracefully shuts down the deduplicator.
	Close() error
}

// RedisDeduplicator implements Deduplicator using Redis Sets.
// It uses a two-level cache:
// - L1: Local bloom filter / map for fast duplicate detection
// - L2: Redis Set for distributed coordination
type RedisDeduplicator struct {
	logger      polylog.Logger
	redisClient redis.UniversalClient
	config      DeduplicatorConfig

	// L1 local cache (map of sessionID -> set of relay hashes)
	localCache   map[string]map[string]struct{}
	localCacheMu sync.RWMutex

	// Key prefix for Redis
	keyPrefix string

	// Lifecycle
	mu       sync.Mutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// DeduplicatorConfig contains configuration for the deduplicator.
type DeduplicatorConfig struct {
	// KeyPrefix is the prefix for Redis keys.
	KeyPrefix string

	// TTLBlocks is how many blocks to keep entries (converted to time).
	TTLBlocks int64

	// BlockTimeSeconds is the assumed block time for TTL calculation.
	BlockTimeSeconds int64

	// LocalCacheSize is the max number of entries in local cache per session.
	// 0 means no local cache.
	LocalCacheSize int

	// CleanupIntervalSeconds is how often to run local cache cleanup.
	CleanupIntervalSeconds int64
}

// NewRedisDeduplicator creates a new Redis-backed deduplicator.
func NewRedisDeduplicator(
	logger polylog.Logger,
	redisClient redis.UniversalClient,
	config DeduplicatorConfig,
) *RedisDeduplicator {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ha:miner:dedup"
	}
	if config.TTLBlocks == 0 {
		config.TTLBlocks = 10 // Default: session length + grace period + buffer
	}
	if config.BlockTimeSeconds == 0 {
		config.BlockTimeSeconds = 6
	}
	if config.CleanupIntervalSeconds == 0 {
		config.CleanupIntervalSeconds = 60
	}

	return &RedisDeduplicator{
		logger:      logging.ForComponent(logger, logging.ComponentDeduplicator),
		redisClient: redisClient,
		config:      config,
		keyPrefix:   config.KeyPrefix,
		localCache:  make(map[string]map[string]struct{}),
	}
}

// Start begins the deduplicator's background processes.
func (d *RedisDeduplicator) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return fmt.Errorf("deduplicator is closed")
	}

	ctx, d.cancelFn = context.WithCancel(ctx)
	d.mu.Unlock()

	// Start local cache cleanup goroutine
	d.wg.Add(1)
	go d.cleanupLoop(ctx)

	d.logger.Info().Msg("deduplicator started")
	return nil
}

// cleanupLoop periodically cleans up expired local cache entries.
func (d *RedisDeduplicator) cleanupLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(time.Duration(d.config.CleanupIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.cleanupLocalCache()
		}
	}
}

// cleanupLocalCache removes sessions with no active entries.
func (d *RedisDeduplicator) cleanupLocalCache() {
	d.localCacheMu.Lock()
	defer d.localCacheMu.Unlock()

	// For now, just clear sessions that are too large
	// A more sophisticated implementation would track timestamps
	for sessionID, hashes := range d.localCache {
		if d.config.LocalCacheSize > 0 && len(hashes) > d.config.LocalCacheSize*2 {
			delete(d.localCache, sessionID)
			d.logger.Debug().
				Str("session_id", sessionID).
				Int("entries", len(hashes)).
				Msg("cleared oversized local cache for session")
		}
	}
}

// IsDuplicate checks if a relay has already been processed.
func (d *RedisDeduplicator) IsDuplicate(ctx context.Context, relayHash []byte, sessionID string) (bool, error) {
	hashKey := hex.EncodeToString(relayHash)

	// Check L1 (local cache) first
	if d.isInLocalCache(sessionID, hashKey) {
		dedupLocalCacheHits.WithLabelValues(sessionID).Inc()
		return true, nil
	}

	// Check L2 (Redis)
	key := d.sessionKey(sessionID)
	exists, err := d.redisClient.SIsMember(ctx, key, hashKey).Result()
	if err != nil {
		dedupErrors.WithLabelValues(sessionID, "redis_check").Inc()
		return false, fmt.Errorf("failed to check Redis: %w", err)
	}

	if exists {
		// Add to local cache for future checks
		d.addToLocalCache(sessionID, hashKey)
		dedupRedisCacheHits.WithLabelValues(sessionID).Inc()
		return true, nil
	}

	dedupMisses.WithLabelValues(sessionID).Inc()
	return false, nil
}

// MarkProcessed marks a relay hash as processed.
func (d *RedisDeduplicator) MarkProcessed(ctx context.Context, relayHash []byte, sessionID string) error {
	hashKey := hex.EncodeToString(relayHash)

	// Add to Redis
	key := d.sessionKey(sessionID)
	ttl := d.getTTL()

	pipe := d.redisClient.Pipeline()
	pipe.SAdd(ctx, key, hashKey)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		dedupErrors.WithLabelValues(sessionID, "redis_mark").Inc()
		return fmt.Errorf("failed to mark processed: %w", err)
	}

	// Add to local cache
	d.addToLocalCache(sessionID, hashKey)

	dedupMarked.WithLabelValues(sessionID).Inc()
	return nil
}

// MarkProcessedBatch marks multiple relay hashes as processed.
func (d *RedisDeduplicator) MarkProcessedBatch(ctx context.Context, relayHashes [][]byte, sessionID string) error {
	if len(relayHashes) == 0 {
		return nil
	}

	key := d.sessionKey(sessionID)
	ttl := d.getTTL()

	// Convert hashes to string keys
	hashKeys := make([]interface{}, len(relayHashes))
	for i, hash := range relayHashes {
		hashKeys[i] = hex.EncodeToString(hash)
	}

	// Add to Redis in batch
	pipe := d.redisClient.Pipeline()
	pipe.SAdd(ctx, key, hashKeys...)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		dedupErrors.WithLabelValues(sessionID, "redis_batch_mark").Inc()
		return fmt.Errorf("failed to mark batch processed: %w", err)
	}

	// Add to local cache
	for _, hashKey := range hashKeys {
		d.addToLocalCache(sessionID, hashKey.(string))
	}

	dedupMarked.WithLabelValues(sessionID).Add(float64(len(relayHashes)))
	return nil
}

// CleanupSession removes all deduplication entries for a session.
func (d *RedisDeduplicator) CleanupSession(ctx context.Context, sessionID string) error {
	// Remove from Redis
	key := d.sessionKey(sessionID)
	err := d.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to cleanup session: %w", err)
	}

	// Remove from local cache
	d.localCacheMu.Lock()
	delete(d.localCache, sessionID)
	d.localCacheMu.Unlock()

	d.logger.Debug().
		Str("session_id", sessionID).
		Msg("cleaned up session deduplication entries")

	return nil
}

// Close gracefully shuts down the deduplicator.
func (d *RedisDeduplicator) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.closed = true

	if d.cancelFn != nil {
		d.cancelFn()
	}

	d.wg.Wait()

	d.logger.Info().Msg("deduplicator closed")
	return nil
}

// sessionKey returns the Redis key for a session's deduplication set.
func (d *RedisDeduplicator) sessionKey(sessionID string) string {
	return d.keyPrefix + ":session:" + sessionID
}

// getTTL returns the TTL for deduplication entries.
func (d *RedisDeduplicator) getTTL() time.Duration {
	return time.Duration(d.config.TTLBlocks*d.config.BlockTimeSeconds) * time.Second
}

// isInLocalCache checks if a hash is in the local cache.
func (d *RedisDeduplicator) isInLocalCache(sessionID, hashKey string) bool {
	d.localCacheMu.RLock()
	defer d.localCacheMu.RUnlock()

	if hashes, ok := d.localCache[sessionID]; ok {
		_, exists := hashes[hashKey]
		return exists
	}
	return false
}

// addToLocalCache adds a hash to the local cache.
func (d *RedisDeduplicator) addToLocalCache(sessionID, hashKey string) {
	d.localCacheMu.Lock()
	defer d.localCacheMu.Unlock()

	if d.localCache[sessionID] == nil {
		d.localCache[sessionID] = make(map[string]struct{})
	}

	// Check size limit
	if d.config.LocalCacheSize > 0 && len(d.localCache[sessionID]) >= d.config.LocalCacheSize {
		// Don't add if at capacity (oldest entries stay, prevents thrashing)
		return
	}

	d.localCache[sessionID][hashKey] = struct{}{}
}

// Verify interface compliance.
var _ Deduplicator = (*RedisDeduplicator)(nil)
