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

const (
	Upgrade_0_1_20_PlanName = "v0.1.20"
)

// generated via: ./tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults
// Once onchain, this can be verified like so:
// pocketd query migration show-morse-claimable-account 0C3B325133D65B6136CD59511CC63F17EF992BE6 --network=main --grpc-insecure=false -o json
var mainNetZeroBalanceMorseClaimableAccountsJSONBZ = []byte(`[
  {
    "morse_src_address": "0C3B325133D65B6136CD59511CC63F17EF992BE6",
    "unstaked_balance": {
      "amount": "0",
      "denom": "upokt"
    },
    "supplier_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "application_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  },
  {
    "morse_src_address": "F022ED4E7CCBCE2ABE54E2E3E51B847247E12DDB",
    "unstaked_balance": {
      "amount": "0",
      "denom": "upokt"
    },
    "supplier_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "application_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  }
]`)

// generated via: ./tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults --testnet
var testNetZeroBalanceMorseClaimableAccountsJSONBZ = []byte(`[
  {
    "morse_src_address": "1C66C4B5905CF32EE9ED9D806D6EE12E93D38C20",
    "unstaked_balance": {
      "amount": "0",
      "denom": "upokt"
    },
    "supplier_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "application_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  },
  {
    "morse_src_address": "1FA385948BFF6856765A048BC9F1920354EF87FD",
    "unstaked_balance": {
      "amount": "0",
      "denom": "upokt"
    },
    "supplier_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "application_stake": {
      "amount": "0",
      "denom": "upokt"
    },
    "claimed_at_height": 0,
    "shannon_dest_address": "",
    "morse_output_address": ""
  }
]`)

// Upgrade_0_1_20 handles the upgrade to release `v0.1.20`.
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
var Upgrade_0_1_20 = Upgrade{
	PlanName: Upgrade_0_1_20_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.19..v0.1.20
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.19..v0.1.20

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
