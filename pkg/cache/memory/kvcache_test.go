package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestMemoryKeyValueCache exercises the basic cache functionality.
func TestMemoryKeyValueCache(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		cache, err := NewKeyValueCache[string]()
		require.NoError(t, err)

		// Test Set and Get
		cache.Set("key1", "value1")
		val, isCached := cache.Get("key1")
		require.True(t, isCached)
		require.Equal(t, "value1", val)

		// Test missing key
		_, isCached = cache.Get("nonexistent")
		require.False(t, isCached)

		// Test Delete
		cache.Delete("key1")
		_, isCached = cache.Get("key1")
		require.False(t, isCached)

		// Test Clear
		cache.Set("key2", "value2")
		cache.Clear()
		_, isCached = cache.Get("key2")
		require.False(t, isCached)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		cache, err := NewKeyValueCache[string](
			WithTTL(100 * time.Millisecond),
		)
		require.NoError(t, err)

		cache.Set("key", "value")

		// Value should be available immediately
		val, isCached := cache.Get("key")
		require.True(t, isCached)
		require.Equal(t, "value", val)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Value should now be expired
		_, isCached = cache.Get("key")
		require.False(t, isCached)
	})

	t.Run("max keys eviction", func(t *testing.T) {
		cache, err := NewKeyValueCache[string](
			WithMaxKeys(2),
			WithEvictionPolicy(FirstInFirstOut),
		)
		require.NoError(t, err)

		// Add values up to max keys
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		// Add one more value, should trigger eviction
		cache.Set("key3", "value3")

		// First value should be evicted
		_, isCached := cache.Get("key1")
		require.False(t, isCached)

		// Other values should still be present
		val, isCached := cache.Get("key2")
		require.True(t, isCached)
		require.Equal(t, "value2", val)

		val, isCached = cache.Get("key3")
		require.True(t, isCached)
		require.Equal(t, "value3", val)
	})
}

// TestKeyValueCache_ErrorCases exercises various error conditions
func TestKeyValueCache_ErrorCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cache, err := NewKeyValueCache[string]()
		require.NoError(t, err)

		// Test with empty key
		cache.Set("", "value")
		val, isCached := cache.Get("")
		require.True(t, isCached)
		require.Equal(t, "value", val)

		// Test with empty value
		cache.Set("key", "")
		val, isCached = cache.Get("key")
		require.True(t, isCached)
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
					cache.Set(key, j)
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
