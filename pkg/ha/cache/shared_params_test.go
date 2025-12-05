//go:build test

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// mockSharedQueryClient implements a basic mock for SharedQueryClient.
type mockSharedQueryClient struct {
	params     *sharedtypes.Params
	queryCount int
}

func (m *mockSharedQueryClient) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	m.queryCount++
	return m.params, nil
}

// mockBlockClient implements a basic mock for BlockClient.
type mockBlockClient struct {
	height int64
}

func (m *mockBlockClient) LastBlock(ctx context.Context) mockBlock {
	return mockBlock{height: m.height}
}

type mockBlock struct {
	height int64
}

func (b mockBlock) Height() int64 {
	return b.height
}

func newTestSharedParamCache(t *testing.T) (*RedisSharedParamCache, *miniredis.Miniredis, *mockSharedQueryClient) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := polyzero.NewLogger()

	mockShared := &mockSharedQueryClient{
		params: &sharedtypes.Params{
			NumBlocksPerSession: 4,
		},
	}

	// We can't use the full constructor since it requires real clients
	// Instead, create a minimal cache for testing
	cache := &RedisSharedParamCache{
		logger:      logger.With("component", "test_cache"),
		redisClient: client,
		config: CacheConfig{
			CachePrefix:      "test:cache",
			TTLBlocks:        1,
			BlockTimeSeconds: 6,
			LockTimeout:      5 * time.Second,
		},
		keys: CacheKeys{Prefix: "test:cache"},
	}

	t.Cleanup(func() {
		cache.Close()
		client.Close()
		mr.Close()
	})

	return cache, mr, mockShared
}

func TestRedisSharedParamCache_CacheKeys(t *testing.T) {
	cache, _, _ := newTestSharedParamCache(t)

	// Test key generation
	// Key format is: {Prefix}:params:shared:{height}
	key := cache.keys.SharedParams(100)
	require.Contains(t, key, "params")
	require.Contains(t, key, "shared")
	require.Contains(t, key, "100")

	// Lock key format is: {Prefix}:lock:params:shared:{height}
	lockKey := cache.keys.SharedParamsLock(100)
	require.Contains(t, lockKey, "lock")
	require.Contains(t, lockKey, "100")
}

func TestRedisSharedParamCache_Close(t *testing.T) {
	cache, _, _ := newTestSharedParamCache(t)

	// Close should not error
	err := cache.Close()
	require.NoError(t, err)

	// Double close should be safe
	err = cache.Close()
	require.NoError(t, err)
}

func TestRedisSharedParamCache_LocalCacheOperations(t *testing.T) {
	cache, _, _ := newTestSharedParamCache(t)

	ctx := context.Background()
	height := int64(100)
	key := cache.keys.SharedParams(height)

	// Initially empty
	_, found := cache.localCache.Load(key)
	require.False(t, found)

	// Store something
	params := &sharedtypes.Params{NumBlocksPerSession: 4}
	cache.localCache.Store(key, params)

	// Should be found
	cached, found := cache.localCache.Load(key)
	require.True(t, found)
	require.Equal(t, params, cached.(*sharedtypes.Params))

	// Test invalidation clears local cache
	cache.localCache.Delete(key)
	_, found = cache.localCache.Load(key)
	require.False(t, found)

	_ = ctx // Keep linter happy
}

func TestCacheConfig_BlocksToTTL(t *testing.T) {
	config := CacheConfig{
		BlockTimeSeconds: 6,
	}

	// 1 block = 6 seconds
	ttl := config.BlocksToTTL(1)
	require.Equal(t, 6*time.Second, ttl)

	// 10 blocks = 60 seconds
	ttl = config.BlocksToTTL(10)
	require.Equal(t, 60*time.Second, ttl)
}
