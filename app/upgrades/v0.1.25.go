package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_25_PlanName = "v0.1.25"
)

// Upgrade_0_1_25 handles the upgrade to release `v0.1.25`.
// This upgrade adds:
// - Reduced SMST / onchain proof size by persisting payload-dehydrated relay responses
// - Reduced event related state bloat by removing unnecessary settlement results from events
// - Updated Morse account recovery allowlist
var Upgrade_0_1_25 = Upgrade{
	PlanName: Upgrade_0_1_25_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.24..v0.1.25
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.24..v0.1.25

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
