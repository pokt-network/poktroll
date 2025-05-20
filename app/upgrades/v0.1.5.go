package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

const Upgrade_0_1_5_PlanName = "v0.1.5"

// Upgrade_0_1_5 handles the upgrade to release `v0.1.5`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the memory footprint when iterating over Suppliers and Applications.
// https://github.com/pokt-network/poktroll/compare/v0.1.4..b92aa0c
var Upgrade_0_1_5 = Upgrade{
	PlanName: Upgrade_0_1_5_PlanName,
	// No state or consensus-breaking changes in this upgrade.
	CreateUpgradeHandler: defaultUpgradeHandler,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
