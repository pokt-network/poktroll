package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
	relayertypes "github.com/pokt-network/poktroll/pkg/relayer/types"
)

// BackupManager handles the backup and restoration of session trees for in-memory storage
type BackupManager struct {
	logger      polylog.Logger
	config      *relayerconfig.RelayMinerSmtBackupConfig
	backupTimer *time.Ticker
	stopChan    chan struct{}
	stopOnce    sync.Once
}

// NewBackupManager creates a new backup manager with the given configuration
func NewBackupManager(
	logger polylog.Logger,
	config *relayerconfig.RelayMinerSmtBackupConfig,
) *BackupManager {
	bm := &BackupManager{
		logger:   logger.With("component", "backup_manager"),
		config:   config,
		stopChan: make(chan struct{}),
	}
	
	// Log the complete backup configuration for debugging
	bm.logBackupConfiguration()
	
	return bm
}

// Start starts the periodic backup process if enabled
func (bm *BackupManager) Start(ctx context.Context, sessionsMgr *relayerSessionsManager) {
	if bm.config == nil || bm.config.IntervalSeconds == 0 {
		bm.logger.Info().Msg("Backup manager disabled or no interval configured")
		return
	}

	bm.logger.Info().
		Uint64("interval_seconds", bm.config.IntervalSeconds).
		Str("backup_dir", bm.config.BackupDir).
		Msg("Starting periodic backup process")

	// Ensure backup directory exists
	if err := os.MkdirAll(bm.config.BackupDir, 0755); err != nil {
		bm.logger.Error().Err(err).Msg("Failed to create backup directory")
		return
	}

	bm.backupTimer = time.NewTicker(time.Duration(bm.config.IntervalSeconds) * time.Second)

	go func() {
		defer bm.backupTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				bm.logger.Info().Msg("Backup manager stopped due to context cancellation")
				return
			case <-bm.stopChan:
				bm.logger.Info().Msg("Backup manager stopped")
				return
			case <-bm.backupTimer.C:
				if err := bm.backupAllSessions(sessionsMgr); err != nil {
					bm.logger.Error().Err(err).Msg("Failed to backup sessions during periodic backup")
				}
			}
		}
	}()
}

// Stop stops the backup manager
func (bm *BackupManager) Stop() {
	bm.logger.Info().Msg("Stopping backup manager")
	bm.stopOnce.Do(func() {
		close(bm.stopChan)
	})
}

// BackupSessionTree backs up a single session tree to disk
func (bm *BackupManager) BackupSessionTree(sessionTreeInterface relayer.SessionTree) error {
	if bm.config == nil {
		return nil
	}

	bm.logger.Debug().
		Str("session_id", sessionTreeInterface.GetSessionHeader().SessionId).
		Str("supplier", sessionTreeInterface.GetSupplierOperatorAddress()).
		Msg("Backing up session tree")

	// Store the SMT root before committing (this is the actual data root we want to preserve)
	smtRoot := sessionTreeInterface.GetSMSTRoot()

	// Create backup data structure using protobuf type
	backupData := &relayertypes.SessionTreeBackupData{
		SessionHeader:           *sessionTreeInterface.GetSessionHeader(),
		SupplierOperatorAddress: sessionTreeInterface.GetSupplierOperatorAddress(),
		ClaimedRoot:             sessionTreeInterface.GetClaimRoot(),
		ProofPath:               sessionTreeInterface.GetProofBz(),
		CompactProofBz:          sessionTreeInterface.GetProofBz(), // Same as ProofPath for now
		IsClaiming:              sessionTreeInterface.IsClaiming(), // Now we can access the actual claiming state
		BackupTimestamp:         time.Now().Unix(),
		// ServiceComputeUnitsPerRelay will be populated during restoration by querying the service
		// We don't populate it here to avoid requiring service query client in backup manager
		ServiceComputeUnitsPerRelay: 0, // Will be filled during restoration
	}

	// Extract SMT data if the KVStore is available (before flushing)
	if kvStore := sessionTreeInterface.GetKVStore(); kvStore != nil {
		// Access the underlying sessionTree to commit SMT data to KVStore
		// This is safe since we're in the same package and the interface is implemented by *sessionTree
		sessionTree := sessionTreeInterface.(*sessionTree)
		if sessionTree.sessionSMT != nil {
			// Commit pending SMT changes to the KVStore before extraction
			if err := sessionTree.sessionSMT.Commit(); err != nil {
				bm.logger.Warn().
					Err(err).
					Str("session_id", sessionTreeInterface.GetSessionHeader().SessionId).
					Msg("Failed to commit SMT before backup - proceeding with metadata-only backup")
			} else {
				// Now extract the committed SMT data from the KVStore
				smtData, err := bm.extractSMTData(kvStore)
				if err != nil {
					bm.logger.Warn().
						Err(err).
						Str("session_id", sessionTreeInterface.GetSessionHeader().SessionId).
						Msg("Failed to extract SMT data - proceeding with metadata-only backup")
				} else {
					backupData.SmtData = smtData
					backupData.SmtRoot = smtRoot // Store the SMT root for proper restoration
					bm.logger.Debug().
						Int("smt_entries", len(smtData)).
						Str("session_id", sessionTreeInterface.GetSessionHeader().SessionId).
						Msg("Successfully extracted SMT data for backup")
				}
			}
		}
	} else {
		bm.logger.Debug().
			Str("session_id", sessionTreeInterface.GetSessionHeader().SessionId).
			Msg("KVStore not available - creating metadata-only backup (session may have been flushed)")
	}

	// Serialize to binary format using protobuf
	backupBytes, err := backupData.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	// Create backup file path with .pb extension for protobuf binary format
	backupFileName := fmt.Sprintf("session_%s_%s_%d.pb",
		sessionTreeInterface.GetSupplierOperatorAddress(),
		sessionTreeInterface.GetSessionHeader().SessionId,
		time.Now().Unix(),
	)
	backupFilePath := filepath.Join(bm.config.BackupDir, backupFileName)

	// Write backup file
	if err := os.WriteFile(backupFilePath, backupBytes, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	bm.logger.Info().
		Str("backup_file", backupFilePath).
		Str("session_id", sessionTreeInterface.GetSessionHeader().SessionId).
		Msg("Session tree backup completed")

	// Cleanup old backups
	if err := bm.cleanupOldBackups(); err != nil {
		bm.logger.Warn().Err(err).Msg("Failed to cleanup old backups")
	}

	return nil
}

// backupAllSessions backs up all active session trees
func (bm *BackupManager) backupAllSessions(sessionsMgr *relayerSessionsManager) error {
	sessionsMgr.sessionsTreesMu.Lock()
	defer sessionsMgr.sessionsTreesMu.Unlock()

	backupCount := 0
	for supplierAddr, heightMap := range sessionsMgr.sessionsTrees {
		for height, sessionMap := range heightMap {
			for sessionId, sessionTree := range sessionMap {
				if err := bm.BackupSessionTree(sessionTree); err != nil {
					bm.logger.Error().
						Err(err).
						Str("supplier", supplierAddr).
						Int64("height", height).
						Str("session_id", sessionId).
						Msg("Failed to backup session tree")
					continue
				}
				backupCount++
			}
		}
	}

	bm.logger.Info().
		Int("backup_count", backupCount).
		Msg("Periodic backup completed")

	return nil
}

// RestoreSessionTrees restores session trees from backup files
func (bm *BackupManager) RestoreSessionTrees() ([]*relayertypes.SessionTreeBackupData, error) {
	if bm.config == nil {
		bm.logger.Info().Msg("Backup manager disabled, skipping restore")
		return nil, nil
	}

	bm.logger.Info().
		Str("backup_dir", bm.config.BackupDir).
		Msg("Starting session tree restoration from backups")

	// Check if backup directory exists
	if _, err := os.Stat(bm.config.BackupDir); os.IsNotExist(err) {
		bm.logger.Info().Msg("Backup directory does not exist, no session trees to restore")
		return nil, nil
	}

	// Read backup directory
	entries, err := os.ReadDir(bm.config.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var restoredSessions []*relayertypes.SessionTreeBackupData
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		// Support only .pb (protobuf binary) files
		if entry.IsDir() || ext != ".pb" {
			continue
		}

		backupFilePath := filepath.Join(bm.config.BackupDir, entry.Name())
		backupData, err := bm.LoadBackupFile(backupFilePath)
		if err != nil {
			bm.logger.Error().
				Err(err).
				Str("backup_file", backupFilePath).
				Msg("Failed to load backup file")
			continue
		}

		restoredSessions = append(restoredSessions, backupData)
	}

	bm.logger.Info().
		Int("restored_sessions", len(restoredSessions)).
		Msg("Session tree restoration completed")

	return restoredSessions, nil
}

// LoadBackupFile loads and parses a protobuf backup file
func (bm *BackupManager) LoadBackupFile(filePath string) (*relayertypes.SessionTreeBackupData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	ext := filepath.Ext(filePath)
	if ext != ".pb" {
		return nil, fmt.Errorf("unsupported backup file format: %s (only .pb files supported)", ext)
	}

	// Handle protobuf binary format
	backupData := new(relayertypes.SessionTreeBackupData)
	if err := backupData.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf backup data: %w", err)
	}
	return backupData, nil
}

// cleanupOldBackups removes old backup files exceeding the retention count
// If RetainBackupCount is 0 or negative, no cleanup is performed (unlimited retention)
func (bm *BackupManager) cleanupOldBackups() error {
	if bm.config.RetainBackupCount <= 0 {
		return nil // No cleanup when 0 or negative (unlimited retention)
	}

	entries, err := os.ReadDir(bm.config.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter backup files (.pb only) and sort by modification time
	var backupFiles []os.DirEntry
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if !entry.IsDir() && ext == ".pb" {
			backupFiles = append(backupFiles, entry)
		}
	}

	if len(backupFiles) <= bm.config.RetainBackupCount {
		return nil // No cleanup needed
	}

	// Sort files by modification time (newest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		infoI, _ := backupFiles[i].Info()
		infoJ, _ := backupFiles[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Remove excess files
	filesToRemove := backupFiles[bm.config.RetainBackupCount:]
	for _, file := range filesToRemove {
		filePath := filepath.Join(bm.config.BackupDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			bm.logger.Warn().
				Err(err).
				Str("file", filePath).
				Msg("Failed to remove old backup file")
		} else {
			bm.logger.Debug().
				Str("file", filePath).
				Msg("Removed old backup file")
		}
	}

	return nil
}

// BackupOnEvent backs up sessions based on configured events
func (bm *BackupManager) BackupOnEvent(
	sessionTree relayer.SessionTree,
	event BackupEvent,
) error {
	if bm.config == nil {
		return nil
	}

	shouldBackup := false
	switch event {
	case BackupEventSessionClose:
		shouldBackup = bm.config.OnSessionClose
	case BackupEventClaimGeneration:
		shouldBackup = bm.config.OnClaimGeneration
	case BackupEventGracefulShutdown:
		shouldBackup = bm.config.OnGracefulShutdown
	}

	if shouldBackup {
		bm.logger.Debug().
			Str("event", string(event)).
			Str("session_id", sessionTree.GetSessionHeader().SessionId).
			Msg("Triggering event-based backup")

		return bm.BackupSessionTree(sessionTree)
	}

	return nil
}

// BackupEvent represents different events that can trigger backups
type BackupEvent string

const (
	BackupEventSessionClose     BackupEvent = "session_close"
	BackupEventClaimGeneration  BackupEvent = "claim_generation"
	BackupEventGracefulShutdown BackupEvent = "graceful_shutdown"
)

// CreateSessionTreeFromBackup reconstructs a session tree from backup data
func CreateSessionTreeFromBackup(
	logger polylog.Logger,
	backupData *relayertypes.SessionTreeBackupData,
	storesDirectoryPath string,
) (*sessionTree, error) {
	logger = logger.With(
		"session_id", backupData.SessionHeader.SessionId,
		"supplier_operator_address", backupData.SupplierOperatorAddress,
	)

	// Create the session tree using the standard constructor
	sessionTreeInterface, err := NewSessionTree(
		logger,
		&backupData.SessionHeader,
		backupData.SupplierOperatorAddress,
		storesDirectoryPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session tree from backup: %w", err)
	}

	sessionTree := sessionTreeInterface.(*sessionTree)

	// Restore additional state from backup
	sessionTree.claimedRoot = backupData.ClaimedRoot
	sessionTree.proofPath = backupData.ProofPath
	sessionTree.compactProofBz = backupData.CompactProofBz
	sessionTree.isClaiming = backupData.IsClaiming

	// Store the backed up SMT root for reconstruction
	var smtRootForReconstruction []byte
	if len(backupData.SmtRoot) > 0 {
		smtRootForReconstruction = backupData.SmtRoot
	} else if len(backupData.ClaimedRoot) > 0 {
		smtRootForReconstruction = backupData.ClaimedRoot
	}

	// Restore SMT data if present in backup
	if len(backupData.SmtData) > 0 {
		// Use proper relay weights from service compute units per relay
		serviceComputeUnits := backupData.ServiceComputeUnitsPerRelay
		if serviceComputeUnits == 0 {
			// Fallback for legacy backups or failed service queries
			serviceComputeUnits = 1
			logger.Warn().Msg("Service compute units per relay not available, using default weight of 1")
		}

		restoredCount, err := restoreKVStoreDataWithCorrectWeights(sessionTree, backupData.SmtData, smtRootForReconstruction, serviceComputeUnits, logger)
		if err != nil {
			logger.Warn().
				Err(err).
				Int("total_entries", len(backupData.SmtData)).
				Msg("Failed to restore some KVStore data entries")
		}

		logger.Info().
			Int("restored_entries", restoredCount).
			Int("total_entries", len(backupData.SmtData)).
			Uint64("service_compute_units", serviceComputeUnits).
			Int64("backup_timestamp", backupData.BackupTimestamp).
			Msg("Session tree restored from backup with SMT data and correct relay weights")
	} else {
		logger.Info().
			Int64("backup_timestamp", backupData.BackupTimestamp).
			Msg("Session tree restored from legacy backup (metadata only)")
	}

	return sessionTree, nil
}

// extractSMTData extracts all key-value pairs from the SMT KVStore
func (bm *BackupManager) extractSMTData(kvStore kvstore.MapStore) ([]*relayertypes.RelayDataEntry, error) {
	// Create iterator to traverse all key-value pairs
	iterator, err := kvStore.NewIterator()
	if err != nil {
		return nil, fmt.Errorf("failed to create SMT iterator: %w", err)
	}
	defer func() {
		if closeErr := iterator.Close(); closeErr != nil {
			bm.logger.Warn().Err(closeErr).Msg("Failed to close SMT iterator during backup")
		}
	}()

	var smtData []*relayertypes.RelayDataEntry

	// Iterate through all key-value pairs
	for iterator.Next() {
		key := iterator.Key()
		value := iterator.Value()

		// Make copies of the key and value to avoid issues with iterator reuse
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)

		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		// Note: We cannot directly extract the weight from the KVStore iterator
		// The weight information is stored internally by the SMST and not accessible
		// through the KVStore interface. For restoration, we'll use a default weight
		// or try to infer it from the relay data if possible.
		entry := &relayertypes.RelayDataEntry{
			Key:    keyCopy,
			Value:  valueCopy,
			Weight: 1, // Default weight - will be updated during restoration if possible
		}

		smtData = append(smtData, entry)
	}

	// Check for iteration errors
	if err := iterator.Error(); err != nil {
		return nil, fmt.Errorf("error during SMT iteration: %w", err)
	}

	return smtData, nil
}

// restoreKVStoreData restores SMT KVStore data directly, preserving the exact tree structure
func restoreKVStoreData(sessionTree *sessionTree, smtData []*relayertypes.RelayDataEntry, smtRoot []byte, logger polylog.Logger) (int, error) {
	if sessionTree.treeStore == nil {
		return 0, fmt.Errorf("session tree KVStore is nil - cannot restore data")
	}

	restoredCount := 0
	var lastError error

	// Directly populate the KVStore with the backed up data
	// This preserves the exact SMT structure including internal nodes
	for _, entry := range smtData {
		err := sessionTree.treeStore.Set(entry.Key, entry.Value)
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed to restore KVStore entry")
			lastError = err
			continue
		}
		restoredCount++
	}

	// After restoring the KVStore data, we need to reconstruct the SMT from it
	// The SMT needs to be rebuilt from the restored KVStore to be functional
	if restoredCount > 0 && len(smtRoot) > 0 {
		// Import the SMT from the restored KVStore data using the backed up SMT root
		sessionTree.sessionSMT = smt.ImportSparseMerkleSumTrie(
			sessionTree.treeStore,
			protocol.NewTrieHasher(),
			smtRoot, // Use the backed up SMT root
			protocol.SMTValueHasher(),
		)
		logger.Debug().
			Str("smt_root", fmt.Sprintf("%x", smtRoot)).
			Msg("Reconstructed SMT from restored KVStore data with backed up root")
	} else if restoredCount > 0 {
		logger.Warn().Msg("SMT root not available for reconstruction - SMT may not function correctly")
	}

	if lastError != nil && restoredCount == 0 {
		return 0, fmt.Errorf("failed to restore any KVStore data: %w", lastError)
	}

	return restoredCount, lastError
}

// restoreKVStoreDataWithCorrectWeights restores SMT KVStore data with proper relay weights
func restoreKVStoreDataWithCorrectWeights(sessionTree *sessionTree, smtData []*relayertypes.RelayDataEntry, smtRoot []byte, serviceComputeUnits uint64, logger polylog.Logger) (int, error) {
	if sessionTree.treeStore == nil {
		return 0, fmt.Errorf("session tree KVStore is nil - cannot restore data")
	}

	restoredCount := 0
	var lastError error

	// First, restore the KVStore data as-is to reconstruct the tree structure
	for _, entry := range smtData {
		err := sessionTree.treeStore.Set(entry.Key, entry.Value)
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("Failed to restore KVStore entry")
			lastError = err
			continue
		}
		restoredCount++
	}

	// After restoring the KVStore data, we need to reconstruct the SMT from it
	if restoredCount > 0 && len(smtRoot) > 0 {
		// Import the SMT from the restored KVStore data using the backed up SMT root
		sessionTree.sessionSMT = smt.ImportSparseMerkleSumTrie(
			sessionTree.treeStore,
			protocol.NewTrieHasher(),
			smtRoot, // Use the backed up SMT root
			protocol.SMTValueHasher(),
		)
		logger.Debug().
			Str("smt_root", fmt.Sprintf("%x", smtRoot)).
			Msg("Reconstructed SMT from restored KVStore data with backed up root")

		// Now we need to fix the relay weights by updating each relay with correct weight
		// This is necessary because the original weights were lost during backup extraction
		for _, entry := range smtData {
			// Skip internal SMT nodes (they typically have specific key patterns)
			// Only update leaf nodes that represent actual relay data
			if len(entry.Key) > 0 && len(entry.Value) > 0 {
				err := sessionTree.sessionSMT.Update(entry.Key, entry.Value, serviceComputeUnits)
				if err != nil {
					logger.Warn().
						Err(err).
						Str("key", fmt.Sprintf("%x", entry.Key)).
						Uint64("weight", serviceComputeUnits).
						Msg("Failed to update relay weight in restored SMT")
					// Don't fail the entire restoration for individual weight update failures
					continue
				}
			}
		}
		
		logger.Info().
			Int("relay_entries", len(smtData)).
			Uint64("corrected_weight", serviceComputeUnits).
			Msg("Successfully corrected relay weights in restored SMT")
	} else if restoredCount > 0 {
		logger.Warn().Msg("SMT root not available for reconstruction - SMT may not function correctly")
	}

	if lastError != nil && restoredCount == 0 {
		return 0, fmt.Errorf("failed to restore any KVStore data: %w", lastError)
	}

	return restoredCount, lastError
}

// restoreSMTData restores SMT data entries into a session tree (legacy method)
// This method is kept for backwards compatibility but is not used in the current implementation
func restoreSMTData(sessionTree *sessionTree, smtData []*relayertypes.RelayDataEntry, logger polylog.Logger) (int, error) {
	restoredCount := 0
	var lastError error

	for _, entry := range smtData {
		// Use the weight from backup, or default to 1 if not available
		weight := entry.Weight
		if weight == 0 {
			weight = 1 // Default weight for legacy entries
		}

		// Update the SMT with the restored data
		// Note: We need to access the underlying SMT directly since the sessionTree interface
		// doesn't expose an Update method for raw key-value pairs
		if sessionTree.sessionSMT != nil {
			err := sessionTree.sessionSMT.Update(entry.Key, entry.Value, weight)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("key", string(entry.Key)).
					Uint64("weight", weight).
					Msg("Failed to restore SMT entry")
				lastError = err
				continue
			}
			restoredCount++
		} else {
			lastError = fmt.Errorf("session SMT is nil - cannot restore data")
			break
		}
	}

	if lastError != nil && restoredCount == 0 {
		return 0, fmt.Errorf("failed to restore any SMT data: %w", lastError)
	}

	return restoredCount, lastError
}

// logBackupConfiguration logs the complete backup configuration for debugging
func (bm *BackupManager) logBackupConfiguration() {
	if bm.config == nil {
		bm.logger.Info().Msg("📦 Backup Configuration: DISABLED (nil config)")
		return
	}

	// Determine backup status
	backupStatus := "DISABLED"
	if bm.config.BackupDir != "" {
		backupStatus = "ENABLED"
	}

	// Determine periodic backup status
	periodicStatus := "DISABLED"
	if bm.config.IntervalSeconds > 0 {
		periodicStatus = fmt.Sprintf("ENABLED (%ds intervals)", bm.config.IntervalSeconds)
	}

	// Count enabled event triggers
	eventTriggers := []string{}
	if bm.config.OnSessionClose {
		eventTriggers = append(eventTriggers, "session_close")
	}
	if bm.config.OnClaimGeneration {
		eventTriggers = append(eventTriggers, "claim_generation")
	}
	if bm.config.OnGracefulShutdown {
		eventTriggers = append(eventTriggers, "graceful_shutdown")
	}

	eventTriggersStr := "NONE"
	if len(eventTriggers) > 0 {
		eventTriggersStr = fmt.Sprintf("%v", eventTriggers)
	}

	// Determine retention policy
	retentionPolicy := "UNLIMITED"
	if bm.config.RetainBackupCount > 0 {
		retentionPolicy = fmt.Sprintf("%d files", bm.config.RetainBackupCount)
	}

	// Log comprehensive backup configuration
	bm.logger.Info().
		Str("backup_status", backupStatus).
		Str("backup_directory", bm.config.BackupDir).
		Str("periodic_backup", periodicStatus).
		Str("event_triggers", eventTriggersStr).
		Str("retention_policy", retentionPolicy).
		Uint64("interval_seconds", bm.config.IntervalSeconds).
		Bool("on_session_close", bm.config.OnSessionClose).
		Bool("on_claim_generation", bm.config.OnClaimGeneration).
		Bool("on_graceful_shutdown", bm.config.OnGracefulShutdown).
		Int("retain_backup_count", bm.config.RetainBackupCount).
		Msg("📦 Backup Configuration Summary")
}
