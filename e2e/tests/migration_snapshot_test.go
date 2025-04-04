//go:build e2e && oneshot && manual

package e2e

import (
	"testing"

	"github.com/regen-network/gocuke"
)

// TestMigrationWithSnapshotData runs the migration_snapshot.feature file ONLY.
// NOTE: This test depends on a large Morse node snapshot being available locally.
// See: https://pocket-snapshot.liquify.com/#/pruned/
//
// To run this test use:
//
//	$ make test_e2e_migration_snapshot
//
// TODO_MAINNET_CRITICAL(@bryanchriswhite): Add an example of how to get the snapshot (e.g. wget ...)
func TestMigrationWithSnapshotData(t *testing.T) {
	gocuke.NewRunner(t, &migrationSuite{}).
		Path("migration_snapshot.feature").
		Run()
}
