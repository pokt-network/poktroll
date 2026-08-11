package app

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file is deliberately an INTERNAL test (package app, not app_test): allUpgrades is
// unexported, and it is precisely the thing that needs asserting.

// TestAllUpgrades_RegistersCurrentUpgrade guards the failure mode that shipped undetected
// into main before v0.1.35: an upgrade descriptor can be fully written, reviewed and
// merged in app/upgrades/, yet never appended to the allUpgrades slice.
//
// Defining the descriptor is not what registers it. setUpgrades ranges over allUpgrades to
// call SetUpgradeHandler, so an unlisted upgrade has no handler. An on-chain plan whose
// name has no registered handler does not degrade to a no-op — x/upgrade panics at the
// upgrade height, halting the chain instead of upgrading it.
//
// Before this test existed, Upgrade_NEXT was referenced only from within its own file and
// nothing caught it.
// The expected plan name is DERIVED from the newest app/upgrades/v*.go file rather than
// hardcoded. A hardcoded name has to be hand-edited by the author of the next release --
// the same manual step this test exists to backstop -- so it would silently stop guarding
// the moment someone forgot. Deriving it means adding vX.Y.Z.go is enough to arm the check.
func TestAllUpgrades_RegistersCurrentUpgrade(t *testing.T) {
	expectedPlanName := newestUpgradePlanName(t)

	require.Contains(t, planNames(t), expectedPlanName,
		"%s is defined in app/upgrades/ but NOT listed in allUpgrades; submitting a %q plan "+
			"would halt the chain at the upgrade height instead of upgrading it. "+
			"Append it to the allUpgrades slice in app/upgrades.go.",
		expectedPlanName+".go", expectedPlanName)
}

// newestUpgradePlanName returns the plan name of the highest-versioned upgrade file in
// app/upgrades/ (e.g. "v0.1.35" for v0.1.35.go).
//
// vNEXT.go and vNEXT_Template.go are excluded on purpose: they are working templates and
// are deliberately never registered.
func newestUpgradePlanName(t *testing.T) string {
	t.Helper()

	entries, err := os.ReadDir("upgrades")
	require.NoError(t, err, "unable to read app/upgrades/")

	var newest string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "v") ||
			!strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") ||
			strings.HasPrefix(name, "vNEXT") {
			continue
		}

		version := strings.TrimSuffix(name, ".go")
		if newest == "" || compareUpgradeVersions(version, newest) > 0 {
			newest = version
		}
	}

	require.NotEmpty(t, newest,
		"found no vX.Y.Z.go upgrade files in app/upgrades/; this test cannot guard registration")
	return newest
}

// compareUpgradeVersions orders two "vX.Y.Z" plan names, returning >0 if a is newer than b.
// A pre-release suffix (e.g. "v0.1.31-beta-2") sorts BELOW the plain release it qualifies.
func compareUpgradeVersions(a, b string) int {
	aBase, aPre, _ := strings.Cut(strings.TrimPrefix(a, "v"), "-")
	bBase, bPre, _ := strings.Cut(strings.TrimPrefix(b, "v"), "-")

	aParts, bParts := strings.Split(aBase, "."), strings.Split(bBase, ".")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		var aNum, bNum int
		if i < len(aParts) {
			aNum, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(bParts[i])
		}
		if aNum != bNum {
			return aNum - bNum
		}
	}

	// Same numeric version: a plain release outranks a pre-release of it.
	switch {
	case aPre == "" && bPre != "":
		return 1
	case aPre != "" && bPre == "":
		return -1
	default:
		return strings.Compare(aPre, bPre)
	}
}

// TestAllUpgrades_HaveWiredHandlers asserts every registered upgrade is actually usable:
// a non-empty plan name and a non-nil handler constructor. setUpgrades dereferences both
// at startup, so a malformed entry is a boot-time failure for every node.
func TestAllUpgrades_HaveWiredHandlers(t *testing.T) {
	for _, upgrade := range allUpgrades {
		require.NotEmpty(t, upgrade.PlanName,
			"every registered upgrade must carry a plan name; an empty name can never match an on-chain plan")
		require.NotNil(t, upgrade.CreateUpgradeHandler,
			"upgrade %q must have a handler constructor", upgrade.PlanName)
	}
}

// TestAllUpgrades_PlanNamesAreUnique asserts no plan name is registered twice.
//
// Upgrade plan names are the key the chain coordinates on. SetUpgradeHandler overwrites
// silently on a duplicate, so a copy-paste that leaves the previous version's name in place
// would replace an earlier handler with a later one rather than failing loudly.
func TestAllUpgrades_PlanNamesAreUnique(t *testing.T) {
	seen := make(map[string]int, len(allUpgrades))
	for _, planName := range planNames(t) {
		seen[planName]++
	}

	for planName, count := range seen {
		require.Equalf(t, 1, count, "plan name %q is registered %d times in allUpgrades", planName, count)
	}
}

// planNames returns the plan name of every registered upgrade.
func planNames(t *testing.T) []string {
	t.Helper()

	names := make([]string, 0, len(allUpgrades))
	for _, upgrade := range allUpgrades {
		names = append(names, upgrade.PlanName)
	}
	return names
}
