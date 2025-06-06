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
	_ "embed"
	"encoding/json"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

//go:embed zero_balance_morse_claimable_accounts.json
var zeroBalanceMorseClaimableAccountsJSONBz []byte

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// This upgrade adds:
// - ...
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

		createZeroBalanceMorseClaimableAccounts := func(ctx context.Context) error {
			var zeroBalanceMorseClaimableAccounts []*migrationtypes.MorseClaimableAccount
			if err := json.Unmarshal(zeroBalanceMorseClaimableAccountsJSONBz, &zeroBalanceMorseClaimableAccounts); err != nil {
				return err
			}

			for _, morseClaimableAccount := range zeroBalanceMorseClaimableAccounts {
				// Ensure that the MorseClaimableAccount DOES NOT exist on-chain (skip if so).
				if _, isFound := keepers.MigrationKeeper.GetMorseClaimableAccount(ctx, morseClaimableAccount.GetMorseSrcAddress()); isFound {
					continue
				}

				// Store the MorseClaimableAccount onchain.
				keepers.MigrationKeeper.SetMorseClaimableAccount(ctx, *morseClaimableAccount)
			}

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := createZeroBalanceMorseClaimableAccounts(ctx); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
