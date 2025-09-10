package session

import (
	"path/filepath"
)

// BackupConfig defines the configuration for session tree backups.
type BackupConfig struct {
	// Enabled determines if backup functionality is active.
	// When false, session trees use in-memory storage only.
	Enabled bool

	// BackupDir is the base directory for backup storage.
	// Each session gets its own subdirectory: {BackupDir}/{supplierAddr}/{sessionId}/
	BackupDir string
}

// DefaultBackupConfig returns a default backup configuration.
func DefaultBackupConfig() BackupConfig {
	return BackupConfig{
		Enabled:   false, // Disabled by default for backward compatibility
		BackupDir: ".poktroll/backups",
	}
}

// GetBackupPath returns the backup storage path for a specific session.
func (c BackupConfig) GetBackupPath(supplierOperatorAddress, sessionId string) string {
	return filepath.Join(c.BackupDir, supplierOperatorAddress, sessionId)
}

// ProductionBackupConfig returns a configuration optimized for production use.
// This enables backup functionality with a standard backup directory.
func ProductionBackupConfig() BackupConfig {
	return BackupConfig{
		Enabled:   true,
		BackupDir: ".poktroll/backups",
	}
}

// HighPerformanceBackupConfig returns a configuration optimized for performance.
// This enables backup functionality with a standard backup directory.
// Performance is now controlled by the BackupKVStore worker pool configuration.
func HighPerformanceBackupConfig() BackupConfig {
	return BackupConfig{
		Enabled:   true,
		BackupDir: ".poktroll/backups",
	}
}