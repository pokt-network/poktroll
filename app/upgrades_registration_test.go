package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/upgrades"
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
func TestAllUpgrades_RegistersCurrentUpgrade(t *testing.T) {
	require.Contains(t, planNames(t), upgrades.Upgrade_0_1_35_PlanName,
		"Upgrade_0_1_35 is defined but NOT listed in allUpgrades; submitting a %q plan "+
			"would halt the chain at the upgrade height instead of upgrading it. "+
			"Append it to the allUpgrades slice in app/upgrades.go.",
		upgrades.Upgrade_0_1_35_PlanName)
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
