package session

import (
	"fmt"
	"sync"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ kvstore.MapStore = (*BackupKVStore)(nil)

// BackupKVStore wraps two KVStores to provide backup functionality.
// 
// Read operations: SYNCHRONOUS - only from primary store for fast performance
// Write operations: SYNCHRONOUS dual writes to both primary and backup stores
//
// This design ensures:
// - Fast relay processing (primary store is typically in-memory)
// - Data durability (backup store provides persistence)
// - Simple and predictable behavior (no async complexity)
// - Lifecycle management (backup can be closed while primary remains active)
type BackupKVStore struct {
	logger polylog.Logger

	// primaryStore is the main store used for all operations requiring immediate responses.
	// Typically fast in-memory storage (e.g., SimpleMap).
	primaryStore kvstore.MapStore

	// backupStore is the secondary store used for durability.
	// Typically persistent disk storage (e.g., Pebble).
	// Can be closed via CloseBackupStore() while primary remains active.
	backupStore kvstore.MapStore

	// Synchronization for backup store access (protects against concurrent close)
	backupMu sync.RWMutex
}

// NewBackupKVStore creates a new BackupKVStore with the given primary and backup stores.
func NewBackupKVStore(
	logger polylog.Logger,
	primaryStore kvstore.MapStore,
	backupStore kvstore.MapStore,
) *BackupKVStore {
	return &BackupKVStore{
		logger:       logger.With("module", "backup_kvstore"),
		primaryStore: primaryStore,
		backupStore:  backupStore,
	}
}

// writeToBackup performs a synchronous write operation to the backup store.
// Thread-safe: Uses read lock to protect against concurrent backup store closure.
func (b *BackupKVStore) writeToBackup(operation func(kvstore.MapStore) error) error {
	b.backupMu.RLock()
	defer b.backupMu.RUnlock()
	
	// Check if backup store has been closed (after claim submission)
	if b.backupStore == nil {
		// Backup store closed - operation is no longer needed
		return nil
	}
	
	return operation(b.backupStore)
}

// Get retrieves a value from the primary store only.
// SYNCHRONOUS operation - returns immediately from in-memory store.
// Never accesses backup store to ensure consistent fast performance.
func (b *BackupKVStore) Get(key []byte) ([]byte, error) {
	return b.primaryStore.Get(key)
}

// Set writes a key-value pair to both stores synchronously.
//
// PRIMARY store: SYNCHRONOUS write - must succeed before continuing
// BACKUP store: SYNCHRONOUS write - written immediately after primary
//
// This ensures Set() provides immediate durability with predictable behavior.
// If either write fails, the entire operation fails.
func (b *BackupKVStore) Set(key, value []byte) error {
	// SYNCHRONOUS: Write to primary store first - MUST succeed
	if err := b.primaryStore.Set(key, value); err != nil {
		return err
	}

	// SYNCHRONOUS: Write to backup store immediately
	return b.writeToBackup(func(backup kvstore.MapStore) error {
		return backup.Set(key, value)
	})
}

// Delete removes a key from both stores synchronously.
//
// PRIMARY store: SYNCHRONOUS delete - must succeed before continuing
// BACKUP store: SYNCHRONOUS delete - removed immediately after primary
//
// This ensures Delete() provides immediate consistency with predictable behavior.
func (b *BackupKVStore) Delete(key []byte) error {
	// SYNCHRONOUS: Delete from primary store first - MUST succeed
	if err := b.primaryStore.Delete(key); err != nil {
		return err
	}

	// SYNCHRONOUS: Delete from backup store immediately
	return b.writeToBackup(func(backup kvstore.MapStore) error {
		return backup.Delete(key)
	})
}

// Len returns the number of entries in the primary store.
// SYNCHRONOUS operation - reads only from primary store for immediate response.
func (b *BackupKVStore) Len() (int, error) {
	return b.primaryStore.Len()
}

// ClearAll removes all entries from both stores synchronously.
//
// PRIMARY store: SYNCHRONOUS clear - must succeed before continuing
// BACKUP store: SYNCHRONOUS clear - cleared immediately after primary
//
// This ensures ClearAll() provides immediate consistency.
func (b *BackupKVStore) ClearAll() error {
	// SYNCHRONOUS: Clear primary store first - MUST succeed
	if err := b.primaryStore.ClearAll(); err != nil {
		return err
	}

	// SYNCHRONOUS: Clear backup store immediately
	return b.writeToBackup(func(backup kvstore.MapStore) error {
		return backup.ClearAll()
	})
}

// RestoreFromBackup populates the primary store from the backup store.
// 
// SYNCHRONOUS OPERATION: Blocks until all data is restored from backup to primary.
// This is typically called during startup/restart before normal operations begin.
// 
// Thread-safety: Acquires backup read lock to prevent interference with backup store closure.
func (b *BackupKVStore) RestoreFromBackup() error {
	logger := b.logger.With("method", "RestoreFromBackup")

	// SYNCHRONOUS: Block and restore all data from backup to primary
	// Read lock protects against concurrent backup store closure
	b.backupMu.RLock()
	defer b.backupMu.RUnlock()

	// Check if backup store has data
	backupLen, err := b.backupStore.Len()
	if err != nil {
		logger.Error().Err(err).Msg("failed to get backup store length")
		return err
	}

	if backupLen == 0 {
		logger.Debug().Msg("backup store is empty, nothing to restore")
		return nil
	}

	logger.Info().
		Int("backup_entries", backupLen).
		Msg("restoring primary store from backup")

	// Try to get all data using GetAll if the backup store supports it
	if pebbleStore, ok := b.backupStore.(interface {
		GetAll(prefixKey []byte, descending bool) (keys, values [][]byte, err error)
	}); ok {
		keys, values, err := pebbleStore.GetAll(nil, false)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get all data from backup store")
			return err
		}

		if len(keys) != len(values) {
			return fmt.Errorf("mismatched keys and values count: %d vs %d", len(keys), len(values))
		}

		// Restore all key-value pairs to primary store
		restoredCount := 0
		for i, key := range keys {
			if err := b.primaryStore.Set(key, values[i]); err != nil {
				logger.Error().
					Err(err).
					Str("key", string(key)).
					Msg("failed to restore key to primary store")
				return err
			}
			restoredCount++
		}

		logger.Info().
			Int("restored_entries", restoredCount).
			Msg("successfully restored all entries from backup to primary store")

		return nil
	}

	// Fallback: if backup store doesn't support GetAll, we can't restore
	logger.Error().Msg("backup store does not support GetAll method, cannot restore")
	return fmt.Errorf("backup store does not support iteration for restoration")
}

// CloseBackupStore closes only the backup store while keeping primary store active.
//
// LIFECYCLE OPERATION: Called after claim submission when backup durability is achieved.
// This closes backup store to free file handles while preserving primary store for proofs.
//
// SYNCHRONOUS OPERATION: Blocks until backup store is closed.
func (b *BackupKVStore) CloseBackupStore() error {
	logger := b.logger.With("method", "CloseBackupStore")
	
	b.backupMu.Lock()
	defer b.backupMu.Unlock()
	
	if b.backupStore == nil {
		logger.Debug().Msg("backup store already closed")
		return nil
	}
	
	// Close the backup store
	if closer, ok := b.backupStore.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			logger.Warn().Err(err).Msg("failed to close backup store")
			// Continue anyway - set to nil to stop further operations
		}
	}
	
	// Set to nil to signal workers that backup is no longer available
	b.backupStore = nil
	
	logger.Info().Msg("backup store closed - primary store remains active for proof generation")
	return nil
}

// Close gracefully shuts down both stores.
//
// SYNCHRONOUS OPERATION: Blocks until both stores are closed.
// This ensures proper cleanup of all resources.
func (b *BackupKVStore) Close() error {
	b.backupMu.Lock()
	defer b.backupMu.Unlock()

	// Close the backup store if it exists and implements io.Closer
	if b.backupStore != nil {
		if closer, ok := b.backupStore.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				b.logger.Warn().Err(err).Msg("failed to close backup store")
			}
		}
		b.backupStore = nil
	}

	// Close the primary store if it implements io.Closer
	if closer, ok := b.primaryStore.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	return nil
}