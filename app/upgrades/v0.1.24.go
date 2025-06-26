package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_24_PlanName = "v0.1.24"
)

// Upgrade_0_1_24 handles the upgrade to release `v0.1.24`.
// This upgrade includes:
// - Supplier query enhancements (dehydrated flag for list-suppliers)
// - Supplier downstaking fixes (funds go to owner address)
// - Session parameter updates (numSuppliersPerSession increased to 50)
// - CLI improvements (count flag for relay command)
// - Telegram bot exchange list updates
var Upgrade_0_1_24 = Upgrade{
	PlanName: Upgrade_0_1_24_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.23..v0.1.24
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.23..v0.1.24

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}