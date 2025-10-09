// DEV_NOTE: This test function MUST be in a separate file so that the following
// build constraint can be applied. This build constraint ensures that features
// which are tagged as @oneshot DO NOT run in CI, or otherwise unexpectedly.
//
// The @oneshot tag indicates that a given feature is non-idempotent with respect
// to its impact on the network state. In such cases, a complete network reset
// is required before running these features again.
//go:build e2e && oneshot

package e2e

import (
	"fmt"
	"testing"

	"github.com/regen-network/gocuke"
)

// TestOneshotTaggedFeatures runs ONLY the features specified by the
// --features-path flag which ARE tagged with the @oneshot tag.
func TestOneshotTaggedFeatures(t *testing.T) {
	// Use migrationSuite which embeds suite, providing access to both
	// regular suite step definitions and migration-specific steps.
	// This is necessary because migration features are tagged with @oneshot.
	gocuke.NewRunner(t, &migrationSuite{}).Path(flagFeaturesPath).
		// ONLY execute features tagged with the @oneshot tag.
		Tags(fmt.Sprintf("%s and not %s", oneshotTag, manualTag)).
		Run()
}
