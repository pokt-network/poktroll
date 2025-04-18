package upgrades

import (
	storetypes "cosmossdk.io/store/types"
)

// Upgrade_0_0_9 is a small upgrade on TestNet.
var Upgrade_0_0_9 = Upgrade{
	PlanName:             "v0.0.9",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
