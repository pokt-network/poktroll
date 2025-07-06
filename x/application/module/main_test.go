package application_test

import (
	"os"
	"testing"
)

// TestMain ensures that tests in this package are run serially
// to avoid race conditions with network resources (leveldb).
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
