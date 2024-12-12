package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestInMemoryCache_NonHistorical tests the basic cache functionality without historical mode
func TestInMemoryCache_NonHistorical(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		cache := NewInMemoryCache[string]()

		// Test Set and Get
		err := cache.Set("key1", "value1")
		require.NoError(t, err)
		val, err := cache.Get("key1")
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		// Test missing key
		_, err = cache.Get("nonexistent")
		require.ErrorIs(t, err, ErrCacheMiss)

		// Test Delete
		cache.Delete("key1")
		_, err = cache.Get("key1")
		require.ErrorIs(t, err, ErrCacheMiss)

		// Test Clear
		err = cache.Set("key2", "value2")
		require.NoError(t, err)
		cache.Clear()
		_, err = cache.Get("key2")
		require.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		cache := NewInMemoryCache[string](
			WithTTL(100 * time.Millisecond),
		)

		err := cache.Set("key", "value")
		require.NoError(t, err)

		// Value should be available immediately
		val, err := cache.Get("key")
		require.NoError(t, err)
		require.Equal(t, "value", val)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Value should now be expired
		_, err = cache.Get("key")
		require.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("max size eviction", func(t *testing.T) {
		cache := NewInMemoryCache[string](
			WithMaxKeys(2),
			WithEvictionPolicy(FirstInFirstOut),
		)

		// Add items up to max size
		err := cache.Set("key1", "value1")
		require.NoError(t, err)
		err = cache.Set("key2", "value2")
		require.NoError(t, err)

		// Add one more item, should trigger eviction
		err = cache.Set("key3", "value3")
		require.NoError(t, err)

		// First item should be evicted
		_, err = cache.Get("key1")
		require.ErrorIs(t, err, ErrCacheMiss)

		// Other items should still be present
		val, err := cache.Get("key2")
		require.NoError(t, err)
		require.Equal(t, "value2", val)

		val, err = cache.Get("key3")
		require.NoError(t, err)
		require.Equal(t, "value3", val)
	})
}

// TestInMemoryCache_Historical tests the historical mode functionality
func TestInMemoryCache_Historical(t *testing.T) {
	t.Run("basic historical operations", func(t *testing.T) {
		cache := NewInMemoryCache[string](
			WithHistoricalMode(100),
		)

		// Test SetAtHeight and GetAtHeight
		err := cache.SetAtHeight("key", "value1", 10)
		require.NoError(t, err)
		err = cache.SetAtHeight("key", "value2", 20)
		require.NoError(t, err)

		// Test getting exact heights
		val, err := cache.GetAtHeight("key", 10)
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		val, err = cache.GetAtHeight("key", 20)
		require.NoError(t, err)
		require.Equal(t, "value2", val)

		// Test getting intermediate height (should return nearest lower height)
		val, err = cache.GetAtHeight("key", 15)
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		// Test getting height before first entry
		_, err = cache.GetAtHeight("key", 5)
		require.ErrorIs(t, err, ErrCacheMiss)

		// Test getting height after last entry
		val, err = cache.GetAtHeight("key", 25)
		require.NoError(t, err)
		require.Equal(t, "value2", val)
	})

	t.Run("historical TTL expiration", func(t *testing.T) {
		cache := NewInMemoryCache[string](
			WithHistoricalMode(100),
			WithTTL(100*time.Millisecond),
		)

		err := cache.SetAtHeight("key", "value1", 10)
		require.NoError(t, err)

		// Value should be available immediately
		val, err := cache.GetAtHeight("key", 10)
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Value should now be expired
		_, err = cache.GetAtHeight("key", 10)
		require.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("pruning old heights", func(t *testing.T) {
		cache := NewInMemoryCache[string](
			WithHistoricalMode(10), // Prune entries older than 10 blocks
		)

		// Add entries at different heights
		err := cache.SetAtHeight("key", "value1", 10)
		require.NoError(t, err)
		err = cache.SetAtHeight("key", "value2", 20)
		require.NoError(t, err)
		err = cache.SetAtHeight("key", "value3", 30)
		require.NoError(t, err)

		// Add a new entry that should trigger pruning
		err = cache.SetAtHeight("key", "value4", 40)
		require.NoError(t, err)

		// Entries more than 10 blocks old should be pruned
		_, err = cache.GetAtHeight("key", 10)
		require.ErrorIs(t, err, ErrCacheMiss)
		_, err = cache.GetAtHeight("key", 20)
		require.ErrorIs(t, err, ErrCacheMiss)

		// Recent entries should still be available
		val, err := cache.GetAtHeight("key", 30)
		require.NoError(t, err)
		require.Equal(t, "value3", val)

		val, err = cache.GetAtHeight("key", 40)
		require.NoError(t, err)
		require.Equal(t, "value4", val)
	})

	t.Run("non-historical operations on historical cache", func(t *testing.T) {
		cache := NewInMemoryCache[string](
			WithHistoricalMode(100),
		)

		// Set some historical values
		err := cache.SetAtHeight("key", "value1", 10)
		require.NoError(t, err)
		err = cache.SetAtHeight("key", "value2", 20)
		require.NoError(t, err)

		// Regular Set should work with latest height
		err = cache.Set("key", "value3")
		require.NoError(t, err)

		// Regular Get should return the latest value
		val, err := cache.Get("key")
		require.NoError(t, err)
		require.Equal(t, "value3", val)

		// Delete should remove all historical values
		cache.Delete("key")
		_, err = cache.GetAtHeight("key", 10)
		require.ErrorIs(t, err, ErrCacheMiss)
		_, err = cache.GetAtHeight("key", 20)
		require.ErrorIs(t, err, ErrCacheMiss)
		_, err = cache.Get("key")
		require.ErrorIs(t, err, ErrCacheMiss)
	})
}

// TestInMemoryCache_ErrorCases tests various error conditions
func TestInMemoryCache_ErrorCases(t *testing.T) {
	t.Run("historical operations on non-historical cache", func(t *testing.T) {
		cache := NewInMemoryCache[string]()

		// Attempting historical operations should return error
		err := cache.SetAtHeight("key", "value", 10)
		require.ErrorIs(t, err, ErrHistoricalModeNotEnabled)

		_, err = cache.GetAtHeight("key", 10)
		require.ErrorIs(t, err, ErrHistoricalModeNotEnabled)
	})

	t.Run("zero values", func(t *testing.T) {
		cache := NewInMemoryCache[string]()

		// Test with empty key
		err := cache.Set("", "value")
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

// TestInMemoryCache_ConcurrentAccess tests thread safety of the cache
func TestInMemoryCache_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent access non-historical", func(t *testing.T) {
		cache := NewInMemoryCache[int]()
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
	})

	t.Run("concurrent access historical", func(t *testing.T) {
		cache := NewInMemoryCache[int](
			WithHistoricalMode(100),
		)
		const numGoroutines = 10
		const numOpsPerGoRoutine = 100

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numOpsPerGoRoutine; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						key := "key"
						err := cache.SetAtHeight(key, j, int64(j))
						require.NoError(t, err)
						_, _ = cache.GetAtHeight(key, int64(j))
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
			t.Errorf("test timed out waiting for goroutines to complete: %+v", ctx.Err())
		case <-done:
			t.Log("test completed successfully")
		}
	})
}
