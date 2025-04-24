package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

const Upgrade_0_1_6_PlanName = "v0.1.6"

// Upgrade_0_1_6 handles the upgrade to release `v0.1.6`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the memory by avoiding unnecessary marshaling of the supplier object
// when iterating over the suppliers.
// https://github.com/pokt-network/poktroll/compare/v0.1.5..c7ab386
var Upgrade_0_1_6 = Upgrade{
	PlanName: Upgrade_0_1_6_PlanName,
	// No state or consensus-breaking changes in this upgrade.
	CreateUpgradeHandler: defaultUpgradeHandler,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
