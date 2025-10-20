package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_30_PlanName = "v0.1.30"
)

// Upgrade_0_1_30 handles the upgrade to release `v0.1.30`.
// This upgrade adds:
// - Fix for supplier service config update logic before activation (issue #1794)
var Upgrade_0_1_30 = Upgrade{
	PlanName: Upgrade_0_1_30_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between vPREV..vNEXT
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
