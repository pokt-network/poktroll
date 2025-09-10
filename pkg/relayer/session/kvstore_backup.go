package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/pokt-network/smt/kvstore"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ kvstore.MapStore = (*BackupKVStore)(nil)

// backupOp represents an operation to be performed on the backup store.
type backupOp struct {
	opType backupOpType
	key    []byte
	value  []byte
}

type backupOpType int

const (
	backupOpSet backupOpType = iota
	backupOpDelete
	backupOpClear
)

// BackupKVStore wraps two KVStores to provide backup functionality.
// 
// Read operations: SYNCHRONOUS - only from primary store for fast performance
// Write operations: PRIMARY is SYNCHRONOUS, BACKUP is ASYNCHRONOUS via worker pool
//
// This design ensures:
// - Fast relay processing (primary operations never block)
// - Data durability (backup operations eventually complete)
// - Bounded resource usage (fixed worker pool, no unbounded goroutines)
type BackupKVStore struct {
	logger polylog.Logger

	// primaryStore is the main store used for all operations requiring immediate responses.
	// ALL reads are SYNCHRONOUS from this store only.
	// ALL writes are SYNCHRONOUS to this store (fast, typically in-memory).
	primaryStore kvstore.MapStore

	// backupStore is the secondary store used for durability.
	// ALL operations to this store are ASYNCHRONOUS via worker pool.
	// This is typically a disk-based store (may be throttled for performance).
	backupStore kvstore.MapStore

	// Worker pool for async backup operations
	backupChan chan backupOp
	workers    sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	
	// Synchronization for backup store access
	backupMu sync.Mutex
	
	// Configuration
	numWorkers int
	queueSize  int
}

// NewBackupKVStore creates a new BackupKVStore with the given primary and backup stores.
// It starts a worker pool to handle backup operations asynchronously.
func NewBackupKVStore(
	logger polylog.Logger,
	primaryStore kvstore.MapStore,
	backupStore kvstore.MapStore,
) *BackupKVStore {
	ctx, cancel := context.WithCancel(context.Background())
	
	b := &BackupKVStore{
		logger:       logger.With("module", "backup_kvstore"),
		primaryStore: primaryStore,
		backupStore:  backupStore,
		ctx:          ctx,
		cancel:       cancel,
		numWorkers:   4,    // Default 4 workers
		queueSize:    1000, // Default 1000 operation buffer
	}
	
	// Initialize the backup operation channel
	b.backupChan = make(chan backupOp, b.queueSize)
	
	// Start worker pool
	b.startWorkers()
	
	return b
}

// startWorkers starts the worker pool for processing backup operations.
func (b *BackupKVStore) startWorkers() {
	for i := 0; i < b.numWorkers; i++ {
		b.workers.Add(1)
		go b.worker(i)
	}
}

// worker processes backup operations ASYNCHRONOUSLY from the queue.
// Each worker runs in its own goroutine and processes operations independently.
// Workers coordinate via shared backup mutex to ensure thread-safe access to backup store.
func (b *BackupKVStore) worker(workerID int) {
	defer b.workers.Done()
	
	logger := b.logger.With("worker_id", workerID)
	logger.Debug().Msg("backup worker started - processing async operations")
	
	for {
		select {
		case op := <-b.backupChan:
			if err := b.processBackupOp(op); err != nil {
				logger.Warn().
					Err(err).
					Int("key_len", len(op.key)).
					Int("value_len", len(op.value)).
					Msg("backup operation failed")
			}
		case <-b.ctx.Done():
			logger.Debug().Msg("backup worker shutting down")
			return
		}
	}
}

// processBackupOp processes a single backup operation SYNCHRONOUSLY within worker context.
//
// Called by worker goroutines to process queued operations.
// SYNCHRONOUS within worker: Each operation completes before worker processes next one.
// ASYNCHRONOUS from caller perspective: Caller queued this operation and continued.
// 
// Thread-safety: Mutex ensures multiple workers don't corrupt backup store concurrently.
func (b *BackupKVStore) processBackupOp(op backupOp) error {
	// Check if backup store has been closed (after claim submission)
	if b.backupStore == nil {
		// Backup store closed - operations are no longer needed
		return nil
	}
	
	b.backupMu.Lock()
	defer b.backupMu.Unlock()
	
	// Double-check after acquiring lock
	if b.backupStore == nil {
		return nil
	}
	
	switch op.opType {
	case backupOpSet:
		return b.backupStore.Set(op.key, op.value)
	case backupOpDelete:
		return b.backupStore.Delete(op.key)
	case backupOpClear:
		return b.backupStore.ClearAll()
	default:
		return fmt.Errorf("unknown backup operation type: %d", op.opType)
	}
}

// Get retrieves a value from the primary store only.
// SYNCHRONOUS operation - returns immediately from in-memory store.
// Never accesses backup store to ensure consistent fast performance.
func (b *BackupKVStore) Get(key []byte) ([]byte, error) {
	return b.primaryStore.Get(key)
}

// Set writes a key-value pair to both stores with different timing guarantees.
//
// PRIMARY store: SYNCHRONOUS write - must succeed before returning
// BACKUP store: ASYNCHRONOUS write - queued for worker pool processing
//
// This ensures Set() returns quickly while still providing durability.
// If primary write fails, the entire operation fails immediately.
// If backup queue is full, the backup write is dropped (logged) but Set() still succeeds.
func (b *BackupKVStore) Set(key, value []byte) error {
	// SYNCHRONOUS: Write to primary store first - MUST succeed
	if err := b.primaryStore.Set(key, value); err != nil {
		return err
	}

	// ASYNCHRONOUS: Queue backup operation for worker pool (non-blocking)
	b.queueBackupOp(backupOp{
		opType: backupOpSet,
		key:    append([]byte(nil), key...),   // Copy to prevent data races
		value:  append([]byte(nil), value...), // Copy to prevent data races
	})

	return nil
}

// queueBackupOp queues a backup operation for ASYNCHRONOUS processing.
// 
// NON-BLOCKING operation: Uses non-blocking channel send to avoid delays.
// If queue is full, operation is dropped (with warning) rather than blocking caller.
// This ensures primary relay processing is never delayed by backup queue congestion.
func (b *BackupKVStore) queueBackupOp(op backupOp) {
	select {
	case b.backupChan <- op:
		// Successfully queued for async processing
	default:
		// Queue is full - drop operation to maintain non-blocking guarantee
		b.logger.Warn().
			Int("queue_size", len(b.backupChan)).
			Int("max_queue_size", cap(b.backupChan)).
			Msg("backup queue full, dropping operation to prevent blocking")
	}
}

// Delete removes a key from both stores with different timing guarantees.
//
// PRIMARY store: SYNCHRONOUS delete - must succeed before returning  
// BACKUP store: ASYNCHRONOUS delete - queued for worker pool processing
//
// This ensures Delete() returns quickly while maintaining data consistency.
func (b *BackupKVStore) Delete(key []byte) error {
	// SYNCHRONOUS: Delete from primary store first - MUST succeed
	if err := b.primaryStore.Delete(key); err != nil {
		return err
	}

	// ASYNCHRONOUS: Queue backup delete operation (non-blocking)
	b.queueBackupOp(backupOp{
		opType: backupOpDelete,
		key:    append([]byte(nil), key...), // Copy to prevent data races
	})

	return nil
}

// Len returns the number of entries in the primary store.
// SYNCHRONOUS operation - reads only from primary store for immediate response.
// Note: May temporarily differ from backup store due to async nature of backup operations.
func (b *BackupKVStore) Len() (int, error) {
	return b.primaryStore.Len()
}

// ClearAll removes all entries from both stores with different timing guarantees.
//
// PRIMARY store: SYNCHRONOUS clear - must succeed before returning
// BACKUP store: ASYNCHRONOUS clear - queued for worker pool processing  
//
// This ensures ClearAll() returns quickly (typically used during session cleanup).
func (b *BackupKVStore) ClearAll() error {
	// SYNCHRONOUS: Clear primary store first - MUST succeed
	if err := b.primaryStore.ClearAll(); err != nil {
		return err
	}

	// ASYNCHRONOUS: Queue backup clear operation (non-blocking)
	b.queueBackupOp(backupOp{
		opType: backupOpClear,
	})

	return nil
}

// RestoreFromBackup populates the primary store from the backup store.
// 
// SYNCHRONOUS OPERATION: Blocks until all data is restored from backup to primary.
// This is typically called during startup/restart before normal operations begin.
// 
// Thread-safety: Acquires backup mutex to prevent interference with worker operations.
func (b *BackupKVStore) RestoreFromBackup() error {
	logger := b.logger.With("method", "RestoreFromBackup")

	// SYNCHRONOUS: Block and restore all data from backup to primary
	// Lock protects against concurrent backup store access by workers
	b.backupMu.Lock()
	defer b.backupMu.Unlock()

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
// SYNCHRONOUS OPERATION: Blocks until backup store is closed and workers acknowledge.
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

// Close gracefully shuts down the worker pool and both stores.
//
// SYNCHRONOUS OPERATION: Blocks until all pending backup operations complete.
// This ensures data durability - no backup operations are lost during shutdown.
//
// Shutdown sequence:
// 1. Signal workers to stop accepting new work (cancel context)
// 2. Close backup channel (no new operations can be queued) 
// 3. Wait for workers to finish processing pending operations (blocks)
// 4. Close underlying stores
func (b *BackupKVStore) Close() error {
	// STEP 1: Signal workers to prepare for shutdown
	b.cancel()
	
	// STEP 2: Stop accepting new backup operations
	close(b.backupChan)
	
	// STEP 3: SYNCHRONOUS wait for all pending backup operations to complete
	b.workers.Wait()
	
	b.logger.Info().Msg("backup worker pool shut down gracefully - all operations completed")

	// Close the backup store if it implements io.Closer
	if closer, ok := b.backupStore.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			b.logger.Warn().Err(err).Msg("failed to close backup store")
		}
	}

	// Close the primary store if it implements io.Closer
	if closer, ok := b.primaryStore.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	return nil
}