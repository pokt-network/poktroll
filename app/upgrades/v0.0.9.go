package upgrades

import storetypes "cosmossdk.io/store/types"

// Upgrade_v0_0_9_2 is an upgrade on Beta TestNet.
var Upgrade_v0_0_9_2 = Upgrade{
	// the transaction needs to have a plan with the same name.
	PlanName:             "v0.0.9-2",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}

// Upgrade_v0_0_9_2 is an upgrade on Beta TestNet.
var Upgrade_v0_0_9_3 = Upgrade{
	// the transaction needs to have a plan with the same name.
	PlanName:             "v0.0.9-3",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
