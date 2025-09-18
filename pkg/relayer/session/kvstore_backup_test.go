package session

import (
	"fmt"
	"testing"

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

		// Verify data exists immediately in internalBackupStore (synchronous writes)
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

		// Verify data is immediately deleted from internalBackupStore (synchronous writes)
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

		// Verify backup store is also cleared immediately (synchronous writes)
		backupLength, err := internalBackupStore.Len()
		require.NoError(t, err)
		require.Equal(t, 0, backupLength)
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
		setErr := internalBackupStore.Set([]byte(k), []byte(v))
		require.NoError(t, setErr)
	}

	// Create empty primary store
	primaryStore := simplemap.NewSimpleMap()
	backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)

	// Restore from backup
	err = backupStore.RestoreFromBackup()
	require.NoError(t, err)

	// Verify all data was restored to primary
	for k, expectedV := range testData {
		value, getErr := primaryStore.Get([]byte(k))
		require.NoError(t, getErr)
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

func TestBackupKVStore_SynchronousOperations(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create stores
	primaryStore := simplemap.NewSimpleMap()
	internalBackupStore := simplemap.NewSimpleMap()
	backupStore := NewBackupKVStore(logger, primaryStore, internalBackupStore)
	defer func() {
		err := backupStore.Close()
		require.NoError(t, err)
	}()

	// Write multiple operations and verify immediate consistency
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

	// Verify all data immediately made it to backup store (synchronous writes)
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

	// Verify deletes immediately made it to backup store (synchronous writes)
	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		_, err := internalBackupStore.Get(key)
		require.Error(t, err) // Should not exist
	}

	// Verify remaining items still exist in both stores
	for i := 10; i < numOps; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		expectedValue := []byte(fmt.Sprintf("value%d", i))
		// Check primary store
		primaryValue, err := primaryStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, expectedValue, primaryValue)

		// Check backup store
		backupValue, err := internalBackupStore.Get(key)
		require.NoError(t, err)
		require.Equal(t, expectedValue, backupValue)
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

	// No delay needed - operations are synchronous

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

	// Verify backup store still has the original data (it was closed, not cleared)
	originalBackupValue, err := backupStore.Get(key)
	require.NoError(t, err)
	require.Equal(t, value, originalBackupValue)

	// Verify backup store does NOT have the new data (it was closed before the new write)
	_, err = backupStore.Get(newKey)
	require.Error(t, err) // Should not exist in backup

	// Verify the BackupKVStore operations after closing backup don't error
	// This tests that closed backup is handled gracefully
	testKey := []byte("test_after_close")
	testValue := []byte("value_after_close")
	err = backupKV.Set(testKey, testValue)
	require.NoError(t, err)

	// Primary should have the new test data
	primaryTestValue, err := primaryStore.Get(testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, primaryTestValue)

	// Backup should NOT have the test data (it's closed)
	_, err = backupStore.Get(testKey)
	require.Error(t, err)

	// Test Delete operation after backup is closed
	err = backupKV.Delete(testKey)
	require.NoError(t, err)

	// Primary should have the key deleted
	_, err = primaryStore.Get(testKey)
	require.Error(t, err)

	// Test ClearAll operation after backup is closed
	err = backupKV.ClearAll()
	require.NoError(t, err)

	// Primary should be empty
	primaryLen, err := primaryStore.Len()
	require.NoError(t, err)
	require.Equal(t, 0, primaryLen)

	// Double-close should not error
	err = backupKV.CloseBackupStore()
	require.NoError(t, err)
}
