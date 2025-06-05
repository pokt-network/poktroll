package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_17_PlanName = "v0.1.17"
)

// Upgrade_0_1_17 handles the upgrade to release `v0.1.17`.
// This upgrade adds:
// - Fix for non-deterministic behavior in the unstaking of Morse suppliers
var Upgrade_0_1_17 = Upgrade{
	PlanName: Upgrade_0_1_17_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.16..v0.1.17

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
