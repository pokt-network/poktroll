package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cache2 "github.com/pokt-network/poktroll/pkg/cache"
)

// TestMemoryHistoricalKeyValueCache exercises the historical key/value cache functionality.
func TestMemoryHistoricalKeyValueCache(t *testing.T) {
	t.Run("basic historical operations", func(t *testing.T) {
		cache, err := NewHistoricalKeyValueCache[string](
			WithHistoricalMode(100),
		)
		require.NoError(t, err)

		// Test SetVersion and GetVersion
		err = cache.SetVersion("key", "value1", 10)
		require.NoError(t, err)

		// Test getting the latest version
		latestVersion, err := cache.getLatestVersionNumber("key")
		require.NoError(t, err)
		require.Equal(t, int64(10), latestVersion)

		// Test getting the latest value
		latestValue, err := cache.GetLatestVersion("key")
		require.NoError(t, err)
		require.Equal(t, "value1", latestValue)

		// Update the latest version
		err = cache.SetVersion("key", "value2", 20)
		require.NoError(t, err)

		// Test getting the latest version
		latestVersion, err = cache.getLatestVersionNumber("key")
		require.NoError(t, err)
		require.Equal(t, int64(20), latestVersion)

		// Test getting the latest value
		latestValue, err = cache.GetLatestVersion("key")
		require.NoError(t, err)
		require.Equal(t, "value2", latestValue)

		// Test getting exact versions
		val, err := cache.GetVersion("key", 10)
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		val, err = cache.GetVersion("key", 20)
		require.NoError(t, err)
		require.Equal(t, "value2", val)

		// Test getting intermediate version (should return nearest lower version)
		val, err = cache.GetVersion("key", 15)
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		// Test getting version before first entry
		_, err = cache.GetVersion("key", 5)
		require.ErrorIs(t, err, cache2.ErrCacheMiss)

		// Test getting version after last entry
		val, err = cache.GetVersion("key", 25)
		require.NoError(t, err)
		require.Equal(t, "value2", val)

		// Test getting a version for a key that isn't cached
		_, err = cache.GetVersion("key2", 20)
		require.ErrorIs(t, err, cache2.ErrCacheMiss)
	})

	t.Run("historical TTL expiration", func(t *testing.T) {
		cache, err := NewHistoricalKeyValueCache[string](
			WithHistoricalMode(100),
			WithTTL(100*time.Millisecond),
		)
		require.NoError(t, err)

		err = cache.SetVersion("key", "value1", 10)
		require.NoError(t, err)

		// Value should be available immediately
		val, err := cache.GetVersion("key", 10)
		require.NoError(t, err)
		require.Equal(t, "value1", val)

		// Wait for ttl to expire
		time.Sleep(150 * time.Millisecond)

		// Value should now be expired
		_, err = cache.GetVersion("key", 10)
		require.ErrorIs(t, err, cache2.ErrCacheMiss)
	})

	t.Run("pruning old versions", func(t *testing.T) {
		cache, err := NewHistoricalKeyValueCache[string](
			WithHistoricalMode(10), // Prune entries older than 10 versions
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
		_, err = cache.GetVersion("key", 10)
		require.ErrorIs(t, err, cache2.ErrCacheMiss)
		_, err = cache.GetVersion("key", 20)
		require.ErrorIs(t, err, cache2.ErrCacheMiss)

		// Recent entries should still be available
		val, err := cache.GetVersion("key", 30)
		require.NoError(t, err)
		require.Equal(t, "value3", val)

		val, err = cache.GetVersion("key", 40)
		require.NoError(t, err)
		require.Equal(t, "value4", val)
	})
}
