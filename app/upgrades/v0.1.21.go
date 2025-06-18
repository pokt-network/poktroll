package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_21_PlanName = "v0.1.21"
)

// Upgrade_0_1_21 handles the upgrade to release `v0.1.21`.
// This upgrade adds:
//  1. Update the recovery allowlist to include the additional accounts
//  2. Slim down excessively sized proof module events:
//     - Changes multiple event protobuf types.
//     - Nodes syncing from genesis will run distinct binaries and swap them at the respective onchain upgrade heights—no state migration required.
//     - WILL impact offchain observers who consume/operate on historical data.
//     - Proper protobuf type (pkg-level) versioning is required to mitigate this.
//     - See: https://github.com/pokt-network/poktroll/issues/1517.
var Upgrade_0_1_21 = Upgrade{
	PlanName: Upgrade_0_1_21_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.20..v0.1.21
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.20..v0.1.21

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
