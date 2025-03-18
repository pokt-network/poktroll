package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

const Upgrade_0_0_14_PlanName = "v0.0.14"

// Upgrade_0_0_14 handles the upgrade to release `v0.0.14`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets.
var Upgrade_0_0_14 = Upgrade{
	PlanName: Upgrade_0_0_14_PlanName,
	// No state changes in this upgrade.
	CreateUpgradeHandler: defaultUpgradeHandler,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
