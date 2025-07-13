package proof_test

import (
	"os"
	"testing"
)

// TestMain configures parallel execution with safety measures.
// Network resources are protected by cross-process file locking.
func TestMain(m *testing.M) {
	// Enable parallel execution within this package - the network package
	// has its own cross-process locking to prevent leveldb race conditions
	os.Exit(m.Run())
}
