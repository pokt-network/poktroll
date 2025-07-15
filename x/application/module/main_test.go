package application_test

import (
	"os"
	"testing"
)

// TestMain ensures that tests in this package are run serially
// to avoid race conditions with network resources (leveldb).
func TestMain(m *testing.M) {
	// Force serial execution for all tests in this package
	// This prevents multiple tests from trying to use network resources simultaneously
	os.Exit(m.Run())
}
