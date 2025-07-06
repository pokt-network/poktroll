package supplier_test

import (
	"os"
	"testing"
)

// TestMain ensures that tests in this package are run serially
// to avoid race conditions with network resources (leveldb).
// This prevents the "panic: leveldb: closed" error that occurs
// when multiple tests try to access the same database concurrently.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
