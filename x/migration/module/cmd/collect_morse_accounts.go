package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	cosmosmath "cosmossdk.io/math"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/signals"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const (
	flagNumAccountsPerDebugLog      = "num-accounts-per-debug-log"
	flagNumAccountsPerDebugLogUsage = "The number of accounts to iterate over for every debug log message that's printed."
)

// A global variable to control the number of accounts to iterate over for every debug log message.
// Used to prevent excessive logging during the account collection process.
var numAccountsPerDebugLog int

// DEV_NOTE: AutoCLI does not apply here because there is no gRPC service, message, or query.
//
// Purpose:
//   - Facilitate deterministic (reproducible) transformation from Morse's export data structure
//     (MorseStateExport) into Shannon's import data structure (MorseAccountState)
//   - Prepare data for use in the MsgImportMorseAccountState message
//
// Note:
// - Does not interact with the network directly
func CollectMorseAccountsCmd() *cobra.Command {
	collectMorseAcctsCmd := &cobra.Command{
		Use:   "collect-morse-accounts [morse-state-export-path] [morse-account-state-path]",
		Args:  cobra.ExactArgs(2),
		Short: "Collect account balances and stakes from [morse-state-export-path] JSON file and output to [morse-account-state-path] as JSON",
		Long: `Processes Morse state for Shannon migration:
- Reads MorseStateExport JSON from [morse-state-export-path]
- Contains account balances and associated stakes
- Outputs MorseAccountState JSON to [morse-account-state-path]
- Integrates with Shannon's MsgUploadMorseState

Generate required input via Morse CLI like so:

	pocket util export-genesis-for-reset [height] [new-chain-id] > morse-state-export.json`,
		RunE:    runCollectMorseAccounts,
		PreRunE: logger.PreRunESetup,
		PostRun: signals.ExitWithCodeIfNonZero,
	}

	collectMorseAcctsCmd.Flags().IntVar(
		&numAccountsPerDebugLog,
		flagNumAccountsPerDebugLog, 0,
		flagNumAccountsPerDebugLogUsage,
	)

	return collectMorseAcctsCmd
}

// runCollectedMorseAccounts is run via the following command:
// $ pocketd migrate collect-morse-accounts
func runCollectMorseAccounts(_ *cobra.Command, args []string) error {
	// DEV_NOTE: No need to check args length due to cobra.ExactArgs(2).
	morseStateExportPath := args[0]
	morseAccountStatePath := args[1]

	logger.Logger.Info().
		Str("morse_state_export_path", morseStateExportPath).
		Str("morse_account_state_path", morseAccountStatePath).
		Msg("collecting Morse accounts...")

	morseWorkspace, err := collectMorseAccounts(morseStateExportPath, morseAccountStatePath)
	if err != nil {
		return err
	}

	return morseWorkspace.infoLogComplete()
}

// collectMorseAccounts:
// - Reads a MorseStateExport JSON file from morseStateExportPath
// - Transforms it into a MorseAccountState
// - Writes the resulting JSON to morseAccountStatePath
func collectMorseAccounts(morseStateExportPath, morseAccountStatePath string) (*morseImportWorkspace, error) {
	if err := validatePathIsFile(morseStateExportPath); err != nil {
		return nil, err
	}

	inputStateJSON, err := os.ReadFile(morseStateExportPath)
	if err != nil {
		return nil, err
	}

	inputState := new(migrationtypes.MorseStateExport)
	if err = cmtjson.Unmarshal(inputStateJSON, inputState); err != nil {
		return nil, err
	}

	morseWorkspace := newMorseImportWorkspace()
	if err = transformMorseState(inputState, morseWorkspace); err != nil {
		return nil, err
	}

	outputStateJSONBz, err := cmtjson.Marshal(morseWorkspace.accountState)
	if err != nil {
		return nil, err
	}

	if err = os.WriteFile(morseAccountStatePath, outputStateJSONBz, 0644); err != nil {
		return nil, err
	}

	return morseWorkspace, nil
}

// validatePathIsFile returns an error if the given path does not exist or is not a file.
func validatePathIsFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return ErrInvalidUsage.Wrapf("[morse-JSON-input-path] cannot be a directory: %s", path)
	}
	return nil
}

// transformMorseState consolidates the Morse account balance, application stake,
// and supplier stake for each account as an entry in the resulting MorseAccountState.
// NOTE: In Shannon terms, a "supplier" is equivalent to all of the following in Morse terms:
// - "validator"
// - "node"
// - "servicer"
func transformMorseState(
	inputState *migrationtypes.MorseStateExport,
	morseWorkspace *morseImportWorkspace,
) error {
	// Iterate over accounts and copy the balances.
	logger.Logger.Info().Msg("collecting account balances...")
	if err := collectInputAccountBalances(inputState, morseWorkspace); err != nil {
		return err
	}

	// Iterate over applications and add the stakes to the corresponding account balances.
	logger.Logger.Info().Msg("collecting application stakes...")
	if err := collectInputApplicationStakes(inputState, morseWorkspace); err != nil {
		return err
	}

	// Iterate over suppliers and add the stakes to the corresponding account balances.
	logger.Logger.Info().Msg("collecting supplier stakes...")
	return collectInputSupplierStakes(inputState, morseWorkspace)
}

// collectInputAccountBalances iterates over the accounts in the inputState and
// adds the balances to the corresponding account balances in the morseWorkspace.
func collectInputAccountBalances(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for exportAccountIdx, exportAccount := range inputState.AppState.Auth.Accounts {
		exportAccountValueJSONBz, err := json.MarshalIndent(exportAccount.Value, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal export account: %w", err)
		}

		// DEV_NOTE: Ignore module accounts.
		// TODO_MAINNET_MIGRATION(@olshansky): Revisit this business logic to ensure that no tokens go missing from Morse to Shannon.
		// See: https://github.com/pokt-network/poktroll/issues/1066 regarding supply validation.
		if exportAccount.Type != "posmint/Account" {
			logger.Logger.Warn().
				Str("type", exportAccount.Type).
				Str("account_json", string(exportAccountValueJSONBz)).
				Msg("ignoring non-EOA account")
			continue
		}

		accountAddr := exportAccount.Value.Address.String()
		if _, _, err := morseWorkspace.addAccount(accountAddr, exportAccount); err != nil {
			return err
		}

		coins := exportAccount.Value.Coins

		// If, for whatever reason, the account has no coins, skip it.
		// DEV_NOTE: This is NEVER expected to happen, but is technically possible.
		if len(coins) == 0 {
			logger.Logger.Warn().Str("address", accountAddr).Msg("account has no coins; skipping")
			continue
		}

		// DEV_NOTE: SHOULD ONLY be one denom (upokt).
		if len(coins) > 1 {
			return ErrMorseExportState.Wrapf(
				"account %q has %d token denominations, expected upokt only: %s",
				accountAddr, len(coins), coins,
			)
		}

		coin := coins[0]
		if coin.Denom != volatile.DenomuPOKT {
			return ErrMorseExportState.Wrapf("unsupported denom %q", coin.Denom)
		}

		if err := morseWorkspace.addUnstakedBalance(accountAddr, coin.Amount); err != nil {
			return fmt.Errorf(
				"adding morse account balance (%s) to account balance of address %q: %w",
				coin, accountAddr, err,
			)
		}

		morseWorkspace.accumulatedTotalBalance = morseWorkspace.accumulatedTotalBalance.Add(coin.Amount)

		if shouldDebugLogProgress(exportAccountIdx) {
			logger.Logger.Debug().
				Int("account_idx", exportAccountIdx).
				Uint64("num_accounts", morseWorkspace.getNumAccounts()).
				Str("total_balance", morseWorkspace.accumulatedTotalBalance.String()).
				Str("grand_total", morseWorkspace.accumulatedTotalsSum().String()).
				Msg("processing account balances...")
		}
	}
	return nil
}

// shouldDebugLogProgress returns true if the given exportAccountIdx should be logged
// via debugLogProgress.
func shouldDebugLogProgress(exportAccountIdx int) bool {
	return numAccountsPerDebugLog > 0 &&
		exportAccountIdx%numAccountsPerDebugLog == 0
}

// collectInputApplicationStakes iterates over the applications in the inputState and
// adds the stake to the corresponding account balances in the morseWorkspace.
func collectInputApplicationStakes(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for exportApplicationIdx, exportApplication := range inputState.AppState.Application.Applications {
		appAddr := exportApplication.Address.String()

		// TODO_MAINNET_MIGRATION(@bryanchriswhite, @olshansk): There are applications
		// present in snapshot data that stakes but no "auth" accounts. Determine:
		// 1. Whether this case is expected or not.
		// 2. What to do about it, if anything.
		if !morseWorkspace.hasAccount(appAddr) {
			logger.Logger.Warn().
				Str("app_address", appAddr).
				Msg("no account found for application")

			// DEV_NOTE: If no auth account was found for this application, create a new one.
			newMorseAppAuthAccount := &migrationtypes.MorseAuthAccount{
				Type: "posmint/Account",
				Value: &migrationtypes.MorseAccount{
					Address: exportApplication.Address,
					Coins:   []cosmostypes.Coin{},
				},
			}
			if _, _, err := morseWorkspace.addAccount(appAddr, newMorseAppAuthAccount); err != nil {
				return fmt.Errorf(
					"adding application account to account balance of address %q: %w",
					appAddr, err,
				)
			}
		}

		if exportApplication.StakedTokens != "" {
			appStakeAmtUpokt, ok := cosmosmath.NewIntFromString(exportApplication.StakedTokens)
			if !ok {
				return ErrMorseExportState.Wrapf("failed to parse application stake amount %q", exportApplication.StakedTokens)
			}

			if err := morseWorkspace.addAppStake(appAddr, appStakeAmtUpokt); err != nil {
				return fmt.Errorf(
					"adding application stake amount to account balance of address %q: %w",
					appAddr, err,
				)
			}

			morseWorkspace.accumulatedTotalAppStake = morseWorkspace.accumulatedTotalAppStake.Add(appStakeAmtUpokt)
			morseWorkspace.numApplications++
		} else {
			exportApplicationJSONBz, err := json.MarshalIndent(exportApplication, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal export supplier: %w", err)
			}

			// CRITICAL: This SHOULD NEVER happen; is indicative of an issue with data deserialization!
			signals.ExitCode += 1
			logger.Logger.Error().
				Str("app_address", appAddr).
				Msgf("account staked as a application but has no stake: %s", string(exportApplicationJSONBz))
		}

		if shouldDebugLogProgress(exportApplicationIdx) {
			logger.Logger.Debug().
				Int("application_idx", exportApplicationIdx).
				Uint64("num_accounts", morseWorkspace.getNumAccounts()).
				Uint64("num_applications", morseWorkspace.numApplications).
				Str("total_app_stake", morseWorkspace.accumulatedTotalAppStake.String()).
				Str("grand_total", morseWorkspace.accumulatedTotalsSum().String()).
				Msg("processing application stakes...")
		}
	}
	return nil
}

// collectInputSupplierStakes iterates over the suppliers in the inputState and
// adds the stake to the corresponding account balances in the morseWorkspace.
func collectInputSupplierStakes(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for exportSupplierIdx, exportSupplier := range inputState.AppState.Pos.Validators {
		supplierAddr := exportSupplier.Address.String()

		// TODO_MAINNET_MIGRATION(@bryanchriswhite, @olshansk): There are suppliers
		// present in snapshot data that stakes but no "auth" accounts. Determine:
		// 1. Whether this case is expected or not.
		// 2. What to do about it, if anything.
		//
		// HYPOTHESIS: One potential explanation for this could be non-custodial
		// supplier stakes, depending on how Morse implemented this feature.
		if !morseWorkspace.hasAccount(supplierAddr) {
			logger.Logger.Warn().
				Str("supplier_address", supplierAddr).
				Msg("no account found for supplier")

			// DEV_NOTE: If no auth account was found for this supplier, create a new one.
			newSupplierAccount := &migrationtypes.MorseAuthAccount{
				Type: "posmint/Account",
				Value: &migrationtypes.MorseAccount{
					Address: exportSupplier.Address,
					Coins:   []cosmostypes.Coin{},
				},
			}
			if _, _, err := morseWorkspace.addAccount(supplierAddr, newSupplierAccount); err != nil {
				return fmt.Errorf(
					"adding supplier account to account balance of address %q: %w",
					supplierAddr, err,
				)
			}
		}

		if exportSupplier.StakedTokens != "" {
			supplierStakeAmtUpokt, ok := cosmosmath.NewIntFromString(exportSupplier.StakedTokens)
			if !ok {
				return ErrMorseExportState.Wrapf("failed to parse supplier stake amount %q", exportSupplier.StakedTokens)
			}

			if err := morseWorkspace.addSupplierStake(supplierAddr, supplierStakeAmtUpokt); err != nil {
				return fmt.Errorf(
					"adding supplier stake amount to account balance of address %q: %w",
					supplierAddr, err,
				)
			}

			morseWorkspace.accumulatedTotalSupplierStake = morseWorkspace.accumulatedTotalSupplierStake.Add(supplierStakeAmtUpokt)
			morseWorkspace.numSuppliers++
		} else {
			exportSupplierJSONBz, err := json.MarshalIndent(exportSupplier, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal export supplier: %w", err)
			}

			// CRITICAL: This SHOULD NEVER happen; is indicative of an issue with data deserialization!
			signals.ExitCode += 1
			logger.Logger.Error().
				Str("supplier_address", supplierAddr).
				Msgf("account staked as a supplier but has no stake: %s", string(exportSupplierJSONBz))
		}

		if shouldDebugLogProgress(exportSupplierIdx) {
			logger.Logger.Debug().
				Int("supplier_idx", exportSupplierIdx).
				Uint64("num_accounts", morseWorkspace.getNumAccounts()).
				Uint64("num_suppliers", morseWorkspace.numSuppliers).
				Str("total_supplier_stake", morseWorkspace.accumulatedTotalSupplierStake.String()).
				Str("grand_total", morseWorkspace.accumulatedTotalsSum().String()).
				Msg("processing accounts...")
		}
	}
	return nil
}
