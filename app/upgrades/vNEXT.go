// vNEXT_Template.go - Canonical Upgrade Template
//
// ────────────────────────────────────────────────────────────────
// TEMPLATE PURPOSE:
//   - This file is the canonical TEMPLATE for all future onchain upgrade files in the poktroll repo.
//   - DO NOT add upgrade-specific logic or changes to this file.
//   - YOU SHOULD NEVER NEED TO CHANGE THIS FILE
//
// USAGE INSTRUCTIONS:
//  1. To start a new upgrade cycle, rename vNEXT.go to the target version (e.g., v0.1.14.go) and update all identifiers accordingly:
//     cp ./app/upgrades/vNEXT.go ./app/upgrades/v0.1.14.go
//  2. Then, copy this file to vNEXT.go:
//     cp ./app/upgrades/vNEXT_Template.go ./app/upgrades/vNEXT.go
//  3. Look for the word "Template" in `vNEXT.go` and replace it with an empty string.
//  4. Make all upgrade-specific changes in vNEXT.go only.
//  5. To reset, restore, or start a new upgrade cycle, repeat fromstep 1.
//  6. Update the last entry in the `allUpgrades` slice in `app/upgrades.go` to point to the new upgrade version variable.
//
// vNEXT_Template.go should NEVER be modified for upgrade-specific logic.
// Only update this file to improve the template itself.
//
//	See also: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT
//
// ────────────────────────────────────────────────────────────────
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
//  1. Update the recovery allowlist to include the additional accounts
//  2. Slim down excessively sized proof module events:
//     - Changes multiple event protobuf types.
//     - Nodes syncing from genesis will run distinct binaries and swap them at the respective onchain upgrade heights—no state migration required.
//     - WILL impact offchain observers who consume/operate on historical data.
//     - Proper protobuf type (pkg-level) versioning is required to mitigate this.
//     - See: https://github.com/pokt-network/poktroll/issues/1517.
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
		// 1. Inspecting the diff between vPREV..vNEXT
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
