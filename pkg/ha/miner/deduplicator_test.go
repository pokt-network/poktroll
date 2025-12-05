package miner

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
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

func TestNewRedisDeduplicator(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	config := DeduplicatorConfig{
		KeyPrefix:      "test:dedup",
		TTLBlocks:      10,
		LocalCacheSize: 1000,
	}

	dedup := NewRedisDeduplicator(logger, client, config)
	require.NotNil(t, dedup)
	require.Equal(t, "test:dedup", dedup.keyPrefix)
	require.Equal(t, int64(10), dedup.config.TTLBlocks)
}

func TestRedisDeduplicator_DefaultConfig(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	config := DeduplicatorConfig{}

	dedup := NewRedisDeduplicator(logger, client, config)
	require.Equal(t, "ha:miner:dedup", dedup.keyPrefix)
	require.Equal(t, int64(10), dedup.config.TTLBlocks)
	require.Equal(t, int64(6), dedup.config.BlockTimeSeconds)
}

func TestRedisDeduplicator_IsDuplicate_NewRelay(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})
	defer dedup.Close()

	ctx := context.Background()
	relayHash := []byte("relay-hash-123")
	sessionID := "session-abc"

	isDup, err := dedup.IsDuplicate(ctx, relayHash, sessionID)
	require.NoError(t, err)
	require.False(t, isDup)
}

func TestRedisDeduplicator_IsDuplicate_AfterMark(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})
	defer dedup.Close()

	ctx := context.Background()
	relayHash := []byte("relay-hash-456")
	sessionID := "session-def"

	// First check - not duplicate
	isDup, err := dedup.IsDuplicate(ctx, relayHash, sessionID)
	require.NoError(t, err)
	require.False(t, isDup)

	// Mark as processed
	err = dedup.MarkProcessed(ctx, relayHash, sessionID)
	require.NoError(t, err)

	// Second check - should be duplicate
	isDup, err = dedup.IsDuplicate(ctx, relayHash, sessionID)
	require.NoError(t, err)
	require.True(t, isDup)
}

func TestRedisDeduplicator_LocalCache(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	config := DeduplicatorConfig{
		LocalCacheSize: 100,
	}

	dedup := NewRedisDeduplicator(logger, client, config)
	defer dedup.Close()

	ctx := context.Background()
	relayHash := []byte("relay-hash-789")
	sessionID := "session-ghi"

	// Mark as processed
	err := dedup.MarkProcessed(ctx, relayHash, sessionID)
	require.NoError(t, err)

	// Check local cache directly
	hashKey := "72656c61792d686173682d373839" // hex of "relay-hash-789"
	require.True(t, dedup.isInLocalCache(sessionID, hashKey))
}

func TestRedisDeduplicator_MarkProcessedBatch(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})
	defer dedup.Close()

	ctx := context.Background()
	sessionID := "session-batch"

	relayHashes := [][]byte{
		[]byte("hash-1"),
		[]byte("hash-2"),
		[]byte("hash-3"),
	}

	// Mark batch as processed
	err := dedup.MarkProcessedBatch(ctx, relayHashes, sessionID)
	require.NoError(t, err)

	// Verify all are marked
	for _, hash := range relayHashes {
		isDup, err := dedup.IsDuplicate(ctx, hash, sessionID)
		require.NoError(t, err)
		require.True(t, isDup)
	}
}

func TestRedisDeduplicator_MarkProcessedBatch_Empty(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})
	defer dedup.Close()

	ctx := context.Background()

	// Empty batch should succeed
	err := dedup.MarkProcessedBatch(ctx, [][]byte{}, "session")
	require.NoError(t, err)
}

func TestRedisDeduplicator_CleanupSession(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{
		LocalCacheSize: 100,
	})
	defer dedup.Close()

	ctx := context.Background()
	sessionID := "session-cleanup"
	relayHash := []byte("hash-to-cleanup")

	// Mark as processed
	err := dedup.MarkProcessed(ctx, relayHash, sessionID)
	require.NoError(t, err)

	// Verify it's marked
	isDup, err := dedup.IsDuplicate(ctx, relayHash, sessionID)
	require.NoError(t, err)
	require.True(t, isDup)

	// Cleanup session
	err = dedup.CleanupSession(ctx, sessionID)
	require.NoError(t, err)

	// Verify it's no longer marked
	isDup, err = dedup.IsDuplicate(ctx, relayHash, sessionID)
	require.NoError(t, err)
	require.False(t, isDup)
}

func TestRedisDeduplicator_DifferentSessions(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})
	defer dedup.Close()

	ctx := context.Background()
	relayHash := []byte("same-hash")
	session1 := "session-1"
	session2 := "session-2"

	// Mark in session 1
	err := dedup.MarkProcessed(ctx, relayHash, session1)
	require.NoError(t, err)

	// Check in session 1 - should be duplicate
	isDup, err := dedup.IsDuplicate(ctx, relayHash, session1)
	require.NoError(t, err)
	require.True(t, isDup)

	// Check in session 2 - should NOT be duplicate
	isDup, err = dedup.IsDuplicate(ctx, relayHash, session2)
	require.NoError(t, err)
	require.False(t, isDup)
}

func TestRedisDeduplicator_Start_AlreadyClosed(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})
	dedup.Close()

	err := dedup.Start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestRedisDeduplicator_Close_Safe(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{})

	// Double close should be safe
	err := dedup.Close()
	require.NoError(t, err)

	err = dedup.Close()
	require.NoError(t, err)
}

func TestRedisDeduplicator_Start_And_Close(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	dedup := NewRedisDeduplicator(logger, client, DeduplicatorConfig{
		CleanupIntervalSeconds: 1, // Fast cleanup for test
	})

	ctx := context.Background()
	err := dedup.Start(ctx)
	require.NoError(t, err)

	// Let cleanup run at least once
	time.Sleep(1500 * time.Millisecond)

	err = dedup.Close()
	require.NoError(t, err)
}

func TestRedisDeduplicator_LocalCacheSizeLimit(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	config := DeduplicatorConfig{
		LocalCacheSize: 3, // Small limit
	}

	dedup := NewRedisDeduplicator(logger, client, config)
	defer dedup.Close()

	ctx := context.Background()
	sessionID := "session-limit"

	// Add more than the limit
	for i := 0; i < 10; i++ {
		hash := []byte{byte(i)}
		err := dedup.MarkProcessed(ctx, hash, sessionID)
		require.NoError(t, err)
	}

	// Local cache should be limited
	dedup.localCacheMu.RLock()
	localSize := len(dedup.localCache[sessionID])
	dedup.localCacheMu.RUnlock()

	// Should be at or near the limit
	require.LessOrEqual(t, localSize, 3)
}

func TestDeduplicatorConfig_Defaults(t *testing.T) {
	config := DeduplicatorConfig{}

	require.Empty(t, config.KeyPrefix)
	require.Equal(t, int64(0), config.TTLBlocks)
	require.Equal(t, int64(0), config.BlockTimeSeconds)
	require.Equal(t, 0, config.LocalCacheSize)
}

func TestRedisDeduplicator_TTL(t *testing.T) {
	_, client := setupMiniredis(t)
	logger := polyzero.NewLogger()

	config := DeduplicatorConfig{
		TTLBlocks:        10,
		BlockTimeSeconds: 6,
	}

	dedup := NewRedisDeduplicator(logger, client, config)

	// 10 blocks * 6 seconds = 60 seconds
	expectedTTL := 60 * time.Second
	require.Equal(t, expectedTTL, dedup.getTTL())
}
