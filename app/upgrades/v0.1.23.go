package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_23_PlanName = "v0.1.23"
)

// Upgrade_0_1_23 handles the upgrade to release `v0.1.23`.
// This upgrade includes:
// - RelayMiner improvements (replaced EventsQueryClient with CometBFT client)
// - Tokenomics enhancements (non-chain halting claim settlement)
// - Service parameter updates and governance parameter adjustments
var Upgrade_0_1_23 = Upgrade{
	PlanName: Upgrade_0_1_23_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.22..v0.1.23
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.22..v0.1.23

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
