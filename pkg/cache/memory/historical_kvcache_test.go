package memory

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/cache"
)

// TestMemoryHistoricalKeyValueCache exercises the historical key/value cache functionality.
func TestMemoryHistoricalKeyValueCache(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		kvcache, err := NewHistoricalKeyValueCache[string](
			WithMaxVersionAge(100),
		)
		require.NoError(t, err)

		// Test SetVersion and GetVersion
		err = kvcache.SetVersion("key", "value1", 10)
		require.NoError(t, err)

		// Test getting the latest version
		latestVersion := kvcache.getLatestVersionNumber("key")
		require.Equal(t, int64(10), latestVersion)

		// Test getting the latest value
		latestValue, isCached := kvcache.GetLatestVersion("key")
		require.True(t, isCached)
		require.Equal(t, "value1", latestValue)

		// Update the latest version
		err = kvcache.SetVersion("key", "value2", 20)
		require.NoError(t, err)

		// Test getting the latest version
		latestVersion = kvcache.getLatestVersionNumber("key")
		require.Equal(t, int64(20), latestVersion)

		// Test getting the latest value
		latestValue, isCached = kvcache.GetLatestVersion("key")
		require.True(t, isCached)
		require.Equal(t, "value2", latestValue)

		// Test getting exact versions
		val, isCached := kvcache.GetVersion("key", 10)
		require.True(t, isCached)
		require.Equal(t, "value1", val)

		val, isCached = kvcache.GetVersion("key", 20)
		require.True(t, isCached)
		require.Equal(t, "value2", val)

		// Test getting intermediate version (should return nearest lower version)
		val, isCached = kvcache.GetVersionLTE("key", 15)
		require.True(t, isCached)
		require.Equal(t, "value1", val)

		// Test getting version before first entry
		_, isCached = kvcache.GetVersion("key", 5)
		require.False(t, isCached)

		// Test getting version after last entry
		val, isCached = kvcache.GetVersionLTE("key", 25)
		require.True(t, isCached)
		require.Equal(t, "value2", val)

		// Test getting a version for a key that isn't cached
		_, isCached = kvcache.GetVersion("key2", 20)
		require.False(t, isCached)

		// Test getting a version for a key that isn't cached
		_, isCached = kvcache.GetVersionLTE("key2", 20)
		require.False(t, isCached)
	})

	t.Run("max keys eviction", func(t *testing.T) {
		kvcache, err := NewHistoricalKeyValueCache[string](WithMaxKeys(2))
		require.NoError(t, err)

		// Add values up to max keys
		err = kvcache.SetVersion("key1", "value1", 10)
		require.NoError(t, err)
		err = kvcache.SetVersion("key2", "value2", 20)
		require.NoError(t, err)

		// Add one more value, should trigger eviction
		err = kvcache.SetVersion("key3", "value3", 30)
		require.NoError(t, err)

		// First value should be evicted
		_, isCached := kvcache.GetVersion("key1", 10)
		require.False(t, isCached)

		// Other values should still be present
		val, isCached := kvcache.GetVersion("key2", 20)
		require.True(t, isCached)
		require.Equal(t, "value2", val)

		val, isCached = kvcache.GetVersion("key3", 30)
		require.True(t, isCached)
		require.Equal(t, "value3", val)
	})

	t.Run("historical cache ignores TTL expiration", func(t *testing.T) {
		cache, err := NewHistoricalKeyValueCache[string](
			WithMaxVersionAge(100),
			WithTTL(100*time.Millisecond),
		)
		require.NoError(t, err)

		err = cache.SetVersion("key", "value1", 10)
		require.NoError(t, err)

		// Value should be available immediately
		val, isCached := cache.GetVersion("key", 10)
		require.True(t, isCached)
		require.Equal(t, "value1", val)

		// Wait for ttl to expire
		time.Sleep(150 * time.Millisecond)

		// Value should now be expired
		_, isCached = cache.GetVersion("key", 10)
		require.True(t, isCached)
	})

	t.Run("pruning old versions", func(t *testing.T) {
		cache, err := NewHistoricalKeyValueCache[string](
			WithMaxVersionAge(10), // Prune entries older than 10 versions
		)
		require.NoError(t, err)

		// Add entries at different versions
		err = cache.SetVersion("key", "value1", 10)
		require.NoError(t, err)
		err = cache.SetVersion("key", "value2", 20)
		require.NoError(t, err)
		err = cache.SetVersion("key", "value3", 30)
		require.NoError(t, err)

		// Add a new entry that should trigger pruning
		err = cache.SetVersion("key", "value4", 40)
		require.NoError(t, err)

		// Entries more than 10 blocks old should be pruned
		_, isCached := cache.GetVersion("key", 10)
		require.False(t, isCached)
		_, isCached = cache.GetVersion("key", 20)
		require.False(t, isCached)

		// Recent entries should still be available
		val, isCached := cache.GetVersion("key", 30)
		require.True(t, isCached)
		require.Equal(t, "value3", val)

		val, isCached = cache.GetVersion("key", 40)
		require.True(t, isCached)
		require.Equal(t, "value4", val)
	})

	t.Run("cannot update existing versions", func(t *testing.T) {
		kvcache, err := NewHistoricalKeyValueCache[string](
			WithMaxVersionAge(100),
		)
		require.NoError(t, err)

		// Add entries at different versions
		err = kvcache.SetVersion("key", "value1", 10)
		require.NoError(t, err)
		err = kvcache.SetVersion("key", "value2", 20)
		require.NoError(t, err)

		// Attempt to update an existing version
		err = kvcache.SetVersion("key", "value3", 10)
		require.EqualError(t, err, cache.ErrNoOverwrite.Wrapf("version: %d", 10).Error())
	})
}

// TestHistoricalKeyValueCache_ConcurrentAccess exercises thread safety of the cache
func TestHistoricalKeyValueCache_ConcurrentAccess(t *testing.T) {
	kvcache, err := NewHistoricalKeyValueCache[int](
		WithMaxVersionAge(10000),
		WithMaxKeys(100),
	)
	require.NoError(t, err)

	const numGoroutines = 10
	const numOpsPerGoRoutine = 100

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoRoutine; j++ {
				// DEV_NOTE: versions MUST be unique per op per goroutine.
				version, _ := strconv.Atoi(fmt.Sprintf("%d%d", i*10, j))

				select {
				case <-ctx.Done():
					return
				default:
					key := "key"
					err = kvcache.SetVersion(key, i+j, int64(version))
					require.NoError(t, err)

					val, isCached := kvcache.GetVersion(key, int64(version))
					require.True(t, isCached)
					require.Equal(t, i+j, val)
				}
			}
		}(i)
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
}
