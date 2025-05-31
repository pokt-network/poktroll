// vNEXT.go - Next Upgrade Placeholder
//
// This file serves as a staging area for the next planned upgrade and contains:
//   - Incremental onchain upgrades specific changes that are not planned for
//     immediate release (e.g. parameter changes, data records restructuring, etc.)
//   - Upgrade handlers and store migrations for the upcoming version
//
// Upgrade Release Process:
// 1. Add any upgrade specific changes in this file until an upgrade is planned
// 2. Once ready for release:
//   - Rename file to the target version (e.g., vNEXT.go → v0.1.14.go)
//   - Change Upgrade_NEXT_PlanName constant to the new version (e.g. Upgrade_v0_1_14_PlanName)
//   - Replace all mentions of "vNEXT" and "vPREV" with appropriate versions
//
// 3. Create a new vNEXT.go file for the subsequent upgrade
package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// This upgrade adds:
// - the `compute_unit_cost_granularity` shared module param
// - the `morse_account_claiming_enabled` migration module param
var Upgrade_NEXT = Upgrade{
	PlanName: Upgrade_NEXT_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between vPREV...vNEXT
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/vPREV...vNEXT

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
