package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_28_PlanName = "v0.1.28"
)

// Upgrade_0_1_28 handles the upgrade to release `v0.1.28`.
// This upgrade adds/changes (diff: v0.1.27..v0.1.28 / current main):
// - Shared module param update: increased `ComputeUnitsPerRelayMax`.
// - Tokenomics updates: validator proper decoding fix; updated DAO address in mint_equals_burn_claim_distribution;
// - Recovery: updated Morse account recovery allowlist (multiple iterations, incl. 8 Aug 2025 update).
var Upgrade_0_1_28 = Upgrade{
	PlanName: Upgrade_0_1_28_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.27..v0.1.28
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.27..v0.1.28

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
