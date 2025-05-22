package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

const (
	Upgrade_0_1_12_PlanName = "v0.1.12"
)

// Upgrade_0_1_12 handles the upgrade to release `v0.1.12`.
// https://github.com/pokt-network/poktroll/compare/v0.1.11..v0.1.12
var Upgrade_0_1_12 = Upgrade{
	PlanName: Upgrade_0_1_12_PlanName,
	// No state or consensus-breaking changes in this upgrade.
	CreateUpgradeHandler: defaultUpgradeHandler,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
