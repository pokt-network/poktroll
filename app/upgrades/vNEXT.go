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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/app/pocket"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

// generated via: ./tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults
// Once onchain, this can be verified like so:
// pocketd query migration show-morse-claimable-account 0C3B325133D65B6136CD59511CC63F17EF992BE6 --network=main --grpc-insecure=false -o json
var mainNetZeroBalanceMorseClaimableAccountsJSONBZ = []byte(`[
  {
    "morse_src_address": "0C3B325133D65B6136CD59511CC63F17EF992BE6",
    "unstaked_balance": "0upokt",
    "supplier_stake": "0upokt",
    "application_stake": "0upokt",
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  },
  {
    "morse_src_address": "F022ED4E7CCBCE2ABE54E2E3E51B847247E12DDB",
    "unstaked_balance": "0upokt",
    "supplier_stake": "0upokt",
    "application_stake": "0upokt",
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  }
]`)

// generated via: ./tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults --testnet
var testNetZeroBalanceMorseClaimableAccountsJSONBZ = []byte(`[
  {
    "morse_src_address": "1C66C4B5905CF32EE9ED9D806D6EE12E93D38C20",
    "unstaked_balance": "0upokt",
    "supplier_stake": "0upokt",
    "application_stake": "0upokt",
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  },
  {
    "morse_src_address": "1FA385948BFF6856765A048BC9F1920354EF87FD",
    "unstaked_balance": "0upokt",
    "supplier_stake": "0upokt",
    "application_stake": "0upokt",
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  }
]`)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// This upgrade adds:
// 1. Creation of zero-balance/stake `MorseClaimableAccount`s for Morse owner accounts that:
//   - Are non-custodial
//   - Had no corresponding `MorseAuthAccount` because they were never used (no balance, no onchain public key)
//   - Were therefore excluded from the canonical `MsgImportMorseClaimableAccounts` import.
//     There is **zero risk** of unintended token minting (staked or unstaked).
//
// 2. Update the Morse account recovery allowlist:
//   - Add all known invalid addresses
//   - Update the exchanges allowlist
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
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			var zeroBalanceMorseClaimableAccounts []*migrationtypes.MorseClaimableAccount

			if err := json.Unmarshal(mainNetZeroBalanceMorseClaimableAccountsJSONBZ, &zeroBalanceMorseClaimableAccounts); err != nil {
				return err
			}

			// For non-main networks, include missing testnet zero-balance morse claimable accounts as well.
			if sdkCtx.ChainID() != pocket.MainNetChainId {
				var testNetZeroBalanceMorseClaimableAccounts []*migrationtypes.MorseClaimableAccount
				if err := json.Unmarshal(testNetZeroBalanceMorseClaimableAccountsJSONBZ, &testNetZeroBalanceMorseClaimableAccounts); err != nil {
					return err
				}

				zeroBalanceMorseClaimableAccounts = append(zeroBalanceMorseClaimableAccounts, testNetZeroBalanceMorseClaimableAccounts...)
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
