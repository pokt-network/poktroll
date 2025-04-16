package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

const Upgrade_0_1_3_PlanName = "v0.1.3"

// Upgrade_0_1_3 handles the upgrade to release `v0.1.3`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets.
// It is a small upgrade intended to reduce the memory footprint of session suppliers.
// Key change:
// - https://github.com/pokt-network/poktroll/pull/1214
// https://github.com/pokt-network/poktroll/compare/v0.1.2..b2d023a
var Upgrade_0_1_3 = Upgrade{
	PlanName: Upgrade_0_1_3_PlanName,
	// No state or consensus-breaking changes in this upgrade.
	CreateUpgradeHandler: defaultUpgradeHandler,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
