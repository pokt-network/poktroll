package network

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// TestLock provides a file-based mutex to coordinate test execution across modules.
// This prevents inter-module race conditions when multiple modules try to use
// network resources (leveldb, ports, etc.) simultaneously.
type TestLock struct {
	lockFile *os.File
	lockPath string
}

// NewTestLock creates a new test lock using a file in the system temp directory.
func NewTestLock(lockName string) (*TestLock, error) {
	tempDir := os.TempDir()
	lockPath := filepath.Join(tempDir, fmt.Sprintf("poktroll_test_%s.lock", lockName))

	// Create or open the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	return &TestLock{
		lockFile: lockFile,
		lockPath: lockPath,
	}, nil
}

// Lock acquires the file lock. This will block until the lock is available.
func (tl *TestLock) Lock() error {
	// Try to acquire an exclusive lock on the file
	for {
		err := syscall.Flock(int(tl.lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Successfully acquired the lock
			return nil
		}
		if err == syscall.EWOULDBLOCK {
			// Lock is held by another process, wait and retry
			time.Sleep(100 * time.Millisecond)
			continue
		}
		// Some other error occurred
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
}

// Unlock releases the file lock.
func (tl *TestLock) Unlock() error {
	err := syscall.Flock(int(tl.lockFile.Fd()), syscall.LOCK_UN)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// Close closes the lock file and cleans up resources.
func (tl *TestLock) Close() error {
	if tl.lockFile != nil {
		err := tl.lockFile.Close()
		if err != nil {
			return fmt.Errorf("failed to close lock file: %w", err)
		}
		// Clean up the lock file
		_ = os.Remove(tl.lockPath)
	}
	return nil
}
