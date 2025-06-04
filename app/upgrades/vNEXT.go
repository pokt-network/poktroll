// vNEXT_.go - Canonical Upgrade
//
// ────────────────────────────────────────────────────────────────
//
//	PURPOSE:
//	 - This file is the canonical  for all future onchain upgrade files in the poktroll repo.
//	 - DO NOT add upgrade-specific logic or changes to this file.
//	 - YOU SHOULD NEVER NEED TO CHANGE THIS FILE
//
// USAGE INSTRUCTIONS:
//  1. To start a new upgrade cycle, copy this file to vNEXT.go:
//     cp ./app/upgrades/vNEXT_.go ./app/upgrades/vNEXT.go
//  2. Look for the word "" in `vNEXT.go` and replace it with an empty string.
//  3. Make all upgrade-specific changes in vNEXT.go only.
//  4. When an upgrade is finalized, rename vNEXT.go to the target version (e.g., v0.1.14.go) and update all identifiers accordingly.
//  5. To reset, restore, or start a new upgrade cycle, repeat step 1.
//
// vNEXT_.go should NEVER be modified for upgrade-specific logic.
// Only update this file to improve the  itself.
//
//	See also: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT
//
// ────────────────────────────────────────────────────────────────
package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// - Normalize Morse accounts recovery allowlist addresses (to uppercase).
// - Normalize Morse source address when handling Morse account recovery message.
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
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger()

			// Adds new authz that were previously incorrect. See #1425
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParams",
				"/pocket.service.MsgRecoverMorseAccount",
			}
			if err := applyNewAuthorizations(ctx, keepers, logger, grantAuthorizationMessages); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
