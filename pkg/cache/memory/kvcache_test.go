package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cache2 "github.com/pokt-network/poktroll/pkg/cache"
)

// TestMemoryKeyValueCache exercises the basic cache functionality.
func TestMemoryKeyValueCache(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		cache, err := NewKeyValueCache[string]()
		require.NoError(t, err)

		// Test Set and Get
		err = cache.Set("key1", "value1")
		require.NoError(t, err)
		val, err := cache.Get("key1")
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		// Test missing key
		_, err = cache.Get("nonexistent")
		require.ErrorIs(t, err, cache2.ErrCacheMiss)

		// Test Delete
		cache.Delete("key1")
		_, err = cache.Get("key1")
		require.ErrorIs(t, err, cache2.ErrCacheMiss)

		// Test Clear
		err = cache.Set("key2", "value2")
		require.NoError(t, err)
		cache.Clear()
		_, err = cache.Get("key2")
		require.ErrorIs(t, err, cache2.ErrCacheMiss)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		cache, err := NewKeyValueCache[string](
			WithTTL(100 * time.Millisecond),
		)
		require.NoError(t, err)

		err = cache.Set("key", "value")
		require.NoError(t, err)

		// Value should be available immediately
		val, err := cache.Get("key")
		require.NoError(t, err)
		require.Equal(t, "value", val)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Value should now be expired
		_, err = cache.Get("key")
		require.ErrorIs(t, err, cache2.ErrCacheMiss)
	})

	t.Run("max keys eviction", func(t *testing.T) {
		cache, err := NewKeyValueCache[string](
			WithMaxKeys(2),
			WithEvictionPolicy(FirstInFirstOut),
		)
		require.NoError(t, err)

		// Add values up to max keys
		err = cache.Set("key1", "value1")
		require.NoError(t, err)
		err = cache.Set("key2", "value2")
		require.NoError(t, err)

		// Add one more value, should trigger eviction
		err = cache.Set("key3", "value3")
		require.NoError(t, err)

		// First value should be evicted
		_, err = cache.Get("key1")
		require.ErrorIs(t, err, cache2.ErrCacheMiss)

		// Other values should still be present
		val, err := cache.Get("key2")
		require.NoError(t, err)
		require.Equal(t, "value2", val)

		val, err = cache.Get("key3")
		require.NoError(t, err)
		require.Equal(t, "value3", val)
	})
}

// TestKeyValueCache_ErrorCases exercises various error conditions
func TestKeyValueCache_ErrorCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cache, err := NewKeyValueCache[string]()
		require.NoError(t, err)

		// Test with empty key
		err = cache.Set("", "value")
		require.NoError(t, err)
		val, err := cache.Get("")
		require.NoError(t, err)
		require.Equal(t, "value", val)

		// Test with empty value
		err = cache.Set("key", "")
		require.NoError(t, err)
		val, err = cache.Get("key")
		require.NoError(t, err)
		require.Equal(t, "", val)
	})
}

// TestKeyValueCache_ConcurrentAccess exercises thread safety of the cache
func TestKeyValueCache_ConcurrentAccess(t *testing.T) {
	cache, err := NewKeyValueCache[int]()
	require.NoError(t, err)

	const numGoroutines = 10
	const numOperations = 100

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					key := "key"
					err := cache.Set(key, j)
					require.NoError(t, err)
					_, _ = cache.Get(key)
				}
			}
		}()
	}

	// Wait for waitgroup with timeout.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		t.Errorf("test timed out waiting for workgroup to complete: %+v", ctx.Err())
	case <-done:
		t.Log("test completed successfully")
	}
}
