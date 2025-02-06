package cache_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/query/cache"
)

func TestKeyValueCache(t *testing.T) {
	kvCache := cache.NewKeyValueCache[any]()

	// Test Get on an empty cache
	_, found := kvCache.Get("key")
	require.False(t, found)

	// Set a value in the cache
	kvCache.Set("key", "value")

	// Test Get on a non-empty cache
	value, found := kvCache.Get("key")
	require.True(t, found)
	require.Equal(t, "value", value)

	// Test Delete on a non-empty cache
	kvCache.Delete("key")

	// Test Get on a deleted key
	_, found = kvCache.Get("key")
	require.False(t, found)

	// Set multiple values in the cache
	kvCache.Set("key1", "value1")
	kvCache.Set("key2", "value2")

	// Test Clear on a non-empty cache
	kvCache.Clear()

	// Test Get on an empty cache
	_, found = kvCache.Get("key1")
	require.False(t, found)

	_, found = kvCache.Get("key2")
	require.False(t, found)

	// Delete a non-existing key
	kvCache.Delete("key1")

	// Test Get on a deleted key
	_, found = kvCache.Get("key1")
	require.False(t, found)

	// Test Clear on an empty cache
	kvCache.Clear()

	// Test Get on an empty cache
	_, found = kvCache.Get("key2")
	require.False(t, found)
}

func TestParamsCache(t *testing.T) {
	paramsCache := cache.NewParamsCache[any]()

	// Test Get on an empty cache
	_, found := paramsCache.Get()
	require.False(t, found)

	// Set a value in the cache
	paramsCache.Set("value")

	// Test Get on a non-empty cache
	value, found := paramsCache.Get()
	require.True(t, found)
	require.Equal(t, "value", value)

	// Test Clear on a non-empty cache
	paramsCache.Clear()

	// Test Get on an empty cache
	_, found = paramsCache.Get()
	require.False(t, found)
}
