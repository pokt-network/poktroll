package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// SessionTreeBackupData represents the serializable data structure for backing up a session tree
type SessionTreeBackupData struct {
	SessionHeader           *sessiontypes.SessionHeader `json:"session_header"`
	SupplierOperatorAddress string                      `json:"supplier_operator_address"`
	ClaimedRoot             []byte                      `json:"claimed_root,omitempty"`
	ProofPath               []byte                      `json:"proof_path,omitempty"`
	CompactProofBz          []byte                      `json:"compact_proof_bz,omitempty"`
	IsClaiming              bool                        `json:"is_claiming"`
	BackupTimestamp         int64                       `json:"backup_timestamp"`
	SMTEntries              []SMTEntry                  `json:"smt_entries"`
}

// SMTEntry represents a key-value pair in the sparse merkle tree
type SMTEntry struct {
	Key    []byte `json:"key"`
	Value  []byte `json:"value"`
	Weight uint64 `json:"weight"`
}

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
	return &BackupManager{
		logger:   logger.With("component", "backup_manager"),
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start starts the periodic backup process if enabled
func (bm *BackupManager) Start(ctx context.Context, sessionsMgr *relayerSessionsManager) {
	if !bm.config.Enabled || bm.config.IntervalSeconds == 0 {
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
func (bm *BackupManager) BackupSessionTree(sessionTree relayer.SessionTree) error {
	if !bm.config.Enabled {
		return nil
	}

	bm.logger.Debug().
		Str("session_id", sessionTree.GetSessionHeader().SessionId).
		Str("supplier", sessionTree.GetSupplierOperatorAddress()).
		Msg("Backing up session tree")

	// Create backup data structure
	backupData := &SessionTreeBackupData{
		SessionHeader:           sessionTree.GetSessionHeader(),
		SupplierOperatorAddress: sessionTree.GetSupplierOperatorAddress(),
		ClaimedRoot:             sessionTree.GetClaimRoot(),
		ProofPath:               sessionTree.GetProofBz(),
		CompactProofBz:          sessionTree.GetProofBz(), // Same as ProofPath for now
		IsClaiming:              false,                    // Cannot access this from interface
		BackupTimestamp:         time.Now().Unix(),
	}

	// Extract SMT entries if the tree is still active (not flushed)
	if sessionTree.GetClaimRoot() == nil {
		entries, err := bm.extractSMTEntries(sessionTree)
		if err != nil {
			return fmt.Errorf("failed to extract SMT entries: %w", err)
		}
		backupData.SMTEntries = entries
	}

	// Serialize to JSON
	backupBytes, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	// Create backup file path
	backupFileName := fmt.Sprintf("session_%s_%s_%d.json",
		sessionTree.GetSupplierOperatorAddress(),
		sessionTree.GetSessionHeader().SessionId,
		time.Now().Unix(),
	)
	backupFilePath := filepath.Join(bm.config.BackupDir, backupFileName)

	// Write backup file
	if err := os.WriteFile(backupFilePath, backupBytes, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	bm.logger.Info().
		Str("backup_file", backupFilePath).
		Str("session_id", sessionTree.GetSessionHeader().SessionId).
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
func (bm *BackupManager) RestoreSessionTrees() ([]*SessionTreeBackupData, error) {
	if !bm.config.Enabled {
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

	var restoredSessions []*SessionTreeBackupData
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		backupFilePath := filepath.Join(bm.config.BackupDir, entry.Name())
		backupData, err := bm.loadBackupFile(backupFilePath)
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

// loadBackupFile loads and parses a backup file
func (bm *BackupManager) loadBackupFile(filePath string) (*SessionTreeBackupData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var backupData SessionTreeBackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	return &backupData, nil
}

// extractSMTEntries extracts all key-value pairs from the SMT
func (bm *BackupManager) extractSMTEntries(sessionTree relayer.SessionTree) ([]SMTEntry, error) {
	// This is a simplified implementation. In practice, you would need to
	// iterate over all entries in the SMT. For now, we'll return an empty
	// slice as the SMT doesn't expose a direct iteration interface.
	//
	// TODO: Implement proper SMT entry extraction when the SMT library
	// provides iteration capabilities or when we can access the underlying KVStore

	bm.logger.Debug().
		Str("session_id", sessionTree.GetSessionHeader().SessionId).
		Msg("SMT entry extraction not yet implemented - storing tree metadata only")

	return []SMTEntry{}, nil
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

	// Filter JSON files and sort by modification time
	var backupFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
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
	if !bm.config.Enabled {
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
	backupData *SessionTreeBackupData,
	storesDirectoryPath string,
) (*sessionTree, error) {
	logger = logger.With(
		"session_id", backupData.SessionHeader.SessionId,
		"supplier_operator_address", backupData.SupplierOperatorAddress,
	)

	// Create the session tree using the standard constructor
	sessionTreeInterface, err := NewSessionTree(
		logger,
		backupData.SessionHeader,
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

	// TODO: Restore SMT entries when SMT library supports bulk loading
	// For now, the tree starts empty and will be populated by new relays
	if len(backupData.SMTEntries) > 0 {
		logger.Debug().
			Int("smt_entries_count", len(backupData.SMTEntries)).
			Msg("SMT entry restoration not yet implemented - tree will start empty")
	}

	logger.Info().
		Int64("backup_timestamp", backupData.BackupTimestamp).
		Msg("Session tree restored from backup")

	return sessionTree, nil
}
