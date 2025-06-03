package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_15_PlanName = "v0.1.15"
)

// Upgrade_0_1_15 handles the upgrade to release `v0.1.15`.
// This upgrade has no state migrations, it only releases new onchain features:
// - Add compute units validation in claim settlement to prevent chain halts when CUPR params change (#1407)
var Upgrade_0_1_15 = Upgrade{
	PlanName: Upgrade_0_1_15_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.14..v0.1.15
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.14..v0.1.15

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
