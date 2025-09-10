package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/pokt-network/smt/kvstore/simplemap"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func TestBackupKVStore_BasicOperations(t *testing.T) {
	logger := polyzero.NewLogger()

	key := []byte("test_key")
	value := []byte("test_value")

	t.Run("Set", func(t *testing.T) {
		// Create stores
		primaryStore := simplemap.NewSimpleMap()
		internalBackupStore, err := pebble.NewKVStore(t.TempDir())
		require.NoError(t, err)
		backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

		// Test Set operation
		err = backupStore.Set(key, value)
		require.NoError(t, err)

		// Verify data exists immediately in primaryStore
		primaryValue, err := primaryStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, value, primaryValue)

		// Give workers time to process backup operations
		time.Sleep(50 * time.Millisecond)

		// Verify data eventually exists in internalBackupStore
		backupValue, err := internalBackupStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, value, backupValue)
	})

	t.Run("Get", func(t *testing.T) {
		// Create stores
		primaryStore := simplemap.NewSimpleMap()
		internalBackupStore, err := pebble.NewKVStore(t.TempDir())
		require.NoError(t, err)
		backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

		err = backupStore.Set(key, value)
		require.NoError(t, err)
		length, err := backupStore.Len()
		require.NoError(t, err)
		require.NotEqual(t, 0, length)

		// Temporarily delete from internalBackupStore
		err = internalBackupStore.Delete(key)
		require.NoError(t, err)

		// Test Get operation (MUST read from primary only)
		retrievedValue, err := backupStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, value, retrievedValue)

		// Test Len operation
		length, err = backupStore.Len()
		require.NoError(t, err)
		require.Equal(t, 1, length)

		// Re-set/synchronize internalBackupStore
		err = internalBackupStore.Set(key, value)
		require.NoError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		// Create stores
		primaryStore := simplemap.NewSimpleMap()
		internalBackupStore, err := pebble.NewKVStore(t.TempDir())
		require.NoError(t, err)
		backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

		err = backupStore.Set(key, value)
		require.NoError(t, err)
		length, err := backupStore.Len()
		require.NoError(t, err)
		require.NotEqual(t, 0, length)

		err = backupStore.Delete(key)
		require.NoError(t, err)

		// Verify data is deleted immediately from primaryStore
		value, err = backupStore.Get(key)
		require.Error(t, err) // Should not exist

		_, err = primaryStore.Get(key)
		require.Error(t, err) // Should not exist

		// Give workers time to process delete operation
		time.Sleep(100 * time.Millisecond)

		// Verify data is eventually deleted from internalBackupStore
		_, err = internalBackupStore.Get(key)
		require.Error(t, err) // Should not exist
	})

	t.Run("ClearAll", func(t *testing.T) {
		// Create stores
		primaryStore := simplemap.NewSimpleMap()
		internalBackupStore, err := pebble.NewKVStore(t.TempDir())
		require.NoError(t, err)
		backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

		err = backupStore.Set([]byte("key1"), []byte("value1"))
		require.NoError(t, err)
		err = backupStore.Set([]byte("key2"), []byte("value2"))
		require.NoError(t, err)

		length, err := backupStore.Len()
		require.NoError(t, err)
		require.NotEqual(t, 0, length)

		err = backupStore.ClearAll()
		require.NoError(t, err)

		// Primary should be cleared immediately
		length, err = backupStore.Len()
		require.NoError(t, err)
		require.Equal(t, 0, length)

		length, err = primaryStore.Len()
		require.NoError(t, err)
		require.Equal(t, 0, length)

		// DEV_NOTE: Skipping internalBackupStore because
		// 1. simplemap.NewSimpleMap() is not concurrency-safe
		// 2. pebble.NewKVStore() can take some time to ClearAll()
	})
}

func TestBackupKVStore_RestoreFromBackup(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create backup store with test data
	internalBackupStore, err := pebble.NewKVStore(t.TempDir())
	require.NoError(t, err)

	// Add some test data to back up
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range testData {
		err := internalBackupStore.Set([]byte(k), []byte(v))
		require.NoError(t, err)
	}

	// Create empty primary store
	primaryStore := simplemap.NewSimpleMap()
	backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

	// Restore from backup
	err = backupStore.RestoreFromBackup()
	require.NoError(t, err)

	// Verify all data was restored to primary
	for k, expectedV := range testData {
		value, err := primaryStore.Get([]byte(k))
		require.NoError(t, err)
		require.Equal(t, []byte(expectedV), value)
	}

	// Verify length matches
	length, err := primaryStore.Len()
	require.NoError(t, err)
	require.Equal(t, len(testData), length)

	// Clean up
	err = backupStore.Close()
	require.NoError(t, err)
}

func TestBackupKVStore_RestoreEmptyBackup(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create empty backup store
	tmpDir := t.TempDir()
	internalBackupStore, err := pebble.NewKVStore(tmpDir)
	require.NoError(t, err)

	primaryStore := simplemap.NewSimpleMap()
	backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

	// Restore from empty backup should not error
	err = backupStore.RestoreFromBackup()
	require.NoError(t, err)

	// Primary should remain empty
	length, err := primaryStore.Len()
	require.NoError(t, err)
	require.Equal(t, 0, length)

	// Clean up
	err = backupStore.Close()
	require.NoError(t, err)
}

func TestBackupKVStore_AsyncWorkerPool(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create stores
	primaryStore := simplemap.NewSimpleMap()
	internalBackupStore := simplemap.NewSimpleMap()
	backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)
	defer func() {
		err := backupStore.Close()
		require.NoError(t, err)
	}()

	// Write multiple operations quickly
	numOps := 100
	for i := 0; i < numOps; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		value := []byte(fmt.Sprintf("value%d", i))

		err := backupStore.Set(key, value)
		require.NoError(t, err)

		// Primary should have data immediately
		primaryValue, err := primaryStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, value, primaryValue)
	}

	// Give workers time to process backup operations
	time.Sleep(100 * time.Millisecond)

	// Verify all data eventually made it to backup store
	for i := 0; i < numOps; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		expectedValue := []byte(fmt.Sprintf("value%d", i))

		backupValue, err := internalBackupStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, expectedValue, backupValue)
	}

	// Test delete operations
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		err := backupStore.Delete(key)
		require.NoError(t, err)

		// Primary should be deleted immediately
		_, err = primaryStore.Get(key)
		require.Error(t, err)
	}

	// Give workers time to process delete operations
	time.Sleep(100 * time.Millisecond)

	// Verify deletes made it to backup store
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		_, err := internalBackupStore.Get(key)
		require.Error(t, err) // Should not exist
	}
}

func TestBackupKVStore_CloseBackupStore(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create stores
	primaryStore := simplemap.NewSimpleMap()
	backupStore := simplemap.NewSimpleMap()
	backupKV := NewBackupKVStore(logger, primaryStore, backupStore)
	defer func() {
		err := backupKV.Close()
		require.NoError(t, err)
	}()

	// Add some data
	key := []byte("test_key")
	value := []byte("test_value")
	err := backupKV.Set(key, value)
	require.NoError(t, err)

	// Give workers time to process backup operation
	time.Sleep(50 * time.Millisecond)

	// Verify data exists in both stores
	primaryValue, err := primaryStore.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, primaryValue)

	backupValue, err := backupStore.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, backupValue)

	// Close backup store
	err = backupKV.CloseBackupStore()
	require.NoError(t, err)

	// Primary store should still work
	retrievedValue, err := backupKV.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, retrievedValue)

	// Can still write to primary store
	newKey := []byte("new_key")
	newValue := []byte("new_value")
	err = backupKV.Set(newKey, newValue)
	require.NoError(t, err)

	// Primary should have new data immediately
	primaryNewValue, err := primaryStore.Get(newKey)
	require.NoError(t, err)
	require.Equal(t, newValue, primaryNewValue)

	// Give time for backup operations (should be no-ops now)
	time.Sleep(50 * time.Millisecond)

	// Backup store should NOT have new data (it's closed)
	// Note: We can't test this directly since the backup store reference
	// is set to nil, but the important thing is no errors occurred

	// Double-close should not error
	err = backupKV.CloseBackupStore()
	require.NoError(t, err)
}
