package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

const Upgrade_0_1_10_PlanName = "v0.1.10"

// Upgrade_0_1_9 handles the upgrade to release `v0.1.9`.
// This is fix upgrade to be issued on both Pocket Network's Shannon Alpha, Beta TestNets.
// It is an upgrade intended to fix chain halts caused by the previous upgrade.
// https://github.com/pokt-network/poktroll/compare/v0.1.9..v0.1.10
var Upgrade_0_1_10 = Upgrade{
	PlanName: Upgrade_0_1_10_PlanName,
	// No state or consensus-breaking changes in this upgrade.
	CreateUpgradeHandler: defaultUpgradeHandler,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
