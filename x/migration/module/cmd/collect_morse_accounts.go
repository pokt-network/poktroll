package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/signals"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const (
	flagNumAccountsPerDebugLog      = "num-accounts-per-debug-log"
	flagNumAccountsPerDebugLogUsage = "The number of accounts to iterate over for every debug log message that's printed."

	flagMergeMorseStateExportPath      = "merge-state"
	flagMergeMorseStateExportPathUsage = "The path to an additional file containing a JSON serialized MorseStateExport object. Used to merge two state export objects (e.g. Morse MainNet and TestNet); useful for testing in Shannon TestNet(s)."
)

var (
	// A global variable to control the number of accounts to iterate over for every debug log message.
	// Used to prevent excessive logging during the account collection process.
	numAccountsPerDebugLog int

	extraMorseStateExportPath string
)

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
		Use:   "collect-morse-accounts [morse-state-export-path] [msg-import-morse-claimable-accounts-path]",
		Args:  cobra.ExactArgs(2),
		Short: "Collect account balances and stakes from [morse-state-export-path] JSON file and output to [msg-import-morse-claimable-accounts-path] as JSON",
		Long: `Processes Morse state for Shannon migration.

This process involves:
	- Reads MorseStateExport JSON from [morse-state-export-path]
	- Contains account balances and associated stakes
	- Outputs MsgMorseImportClaimableAccount JSON to [msg-import-morse-claimable-accounts-path]

The required input MUST be generated via the Morse CLI like so:

	pocket util export-genesis-for-reset [height] pocket > morse-state-export.json

See: https://dev.poktroll.com/operate/morse_migration/state_transfer_playbook
`,
		Example: `pocketd tx migration collect-morse-accounts "$MORSE_STATE_EXPORT_PATH" "$MSG_IMPORT_MORSE_ACCOUNTS_PATH"
pocketd tx migration collect-morse-accounts "$MORSE_STATE_EXPORT_PATH" "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --merge-state="$MORSE_TESTNET_STATE_EXPORT_PATH"
`,
		RunE:    runCollectMorseAccounts,
		PostRun: signals.ExitWithCodeIfNonZero,
	}

	collectMorseAcctsCmd.Flags().IntVar(
		&numAccountsPerDebugLog,
		flagNumAccountsPerDebugLog, 0,
		flagNumAccountsPerDebugLogUsage,
	)

	collectMorseAcctsCmd.Flags().StringVar(
		&extraMorseStateExportPath,
		flagMergeMorseStateExportPath, "",
		flagMergeMorseStateExportPathUsage,
	)

	return collectMorseAcctsCmd
}

// runCollectedMorseAccounts is run via the following command:
// $ pocketd tx migration collect-morse-accounts
func runCollectMorseAccounts(_ *cobra.Command, args []string) error {
	// DEV_NOTE: No need to check args length due to cobra.ExactArgs(2).
	morseStateExportPath := args[0]
	msgImportMorseClaimableAccountsPath := args[1]

	logger.Logger.Info().
		Str("morse_state_export_path", morseStateExportPath).
		Str("msg_import_morse_claimable_accounts_path", msgImportMorseClaimableAccountsPath).
		Msg("collecting Morse accounts...")

	morseWorkspace, err := collectMorseAccounts(morseStateExportPath, msgImportMorseClaimableAccountsPath)
	if err != nil {
		return err
	}

	return morseWorkspace.infoLogComplete()
}

// collectMorseAccounts:
// - Reads a MorseStateExport JSON file from morseStateExportPath
// - Transforms it into a MorseAccountState
// - Writes the resulting JSON to morseAccountStatePath
func collectMorseAccounts(morseStateExportPath, msgImportMorseClaimableAccountsPath string) (*morseImportWorkspace, error) {
	if err := validatePathIsFile(morseStateExportPath); err != nil {
		return nil, err
	}

	// Check if the extra state export path is set and valid.
	if extraMorseStateExportPath != "" {
		if err := validatePathIsFile(extraMorseStateExportPath); err != nil {
			return nil, err
		}
	}

	morseStateExport, err := loadMorseState(morseStateExportPath)
	if err != nil {
		return nil, err
	}

	morseWorkspace := newMorseImportWorkspace()
	if err = transformAndIncludeMorseState(morseStateExport, morseWorkspace); err != nil {
		return nil, err
	}

	// If an extra state export path is provided, include it in the output MorseAccountState.
	if extraMorseStateExportPath != "" {
		extraMorseStateExport, loadErr := loadMorseState(extraMorseStateExportPath)
		if loadErr != nil {
			return nil, loadErr
		}

		if err = transformAndIncludeMorseState(extraMorseStateExport, morseWorkspace); err != nil {
			return nil, err
		}
	}

	msgImportMorseClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*morseWorkspace.accountState,
	)
	if err != nil {
		return nil, err
	}

	msgImportMorseClaimableAccountsJSONBz, err := cmtjson.MarshalIndent(msgImportMorseClaimableAccounts, "", "  ")
	if err != nil {
		return nil, err
	}

	if err = os.WriteFile(msgImportMorseClaimableAccountsPath, msgImportMorseClaimableAccountsJSONBz, 0644); err != nil {
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

// transformAndIncludeMorseState consolidates the Morse account balance, application stake,
// and supplier stake for each account as an entry in the resulting MorseAccountState.
// NOTE: In Shannon terms, a "supplier" is equivalent to all of the following in Morse terms:
// - "validator"
// - "node"
// - "servicer"
func transformAndIncludeMorseState(
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
	for exportAccountIdx, exportAuthAccount := range inputState.AppState.Auth.Accounts {
		exportAuthAccountJSONBz, err := json.MarshalIndent(exportAuthAccount.Value, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal export account: %w", err)
		}

		// DEV_NOTE: Use the module account name as the MorseClaimableAccount address
		// to make it easier to identify downstream.
		var (
			exportAccount *migrationtypes.MorseAccount
			accountAddr   string
		)
		switch exportAuthAccount.Type {
		case migrationtypes.MorseExternallyOwnedAccountType:
			exportAccount, err = exportAuthAccount.AsMorseAccount()
			if err != nil {
				return err
			}

			accountAddr = exportAccount.Address.String()
		case migrationtypes.MorseModuleAccountType:
			// Exclude stake pool module accounts from the MorseAccountState.
			exportModuleAccount, moduleAcctErr := exportAuthAccount.AsMorseModuleAccount()
			if moduleAcctErr != nil {
				return moduleAcctErr
			}
			switch exportModuleAccount.GetName() {
			case migrationtypes.MorseModuleAccountNameApplicationStakeTokensPool,
				migrationtypes.MorseModuleAccountNameStakedTokensPool:
				continue
			}

			exportAccount = &exportModuleAccount.BaseAccount
			accountAddr = exportModuleAccount.GetName()
		default:
			logger.Logger.Warn().
				Str("type", exportAuthAccount.Type).
				Str("account_json", string(exportAuthAccountJSONBz)).
				Msg("ignoring unknown account type")
			continue
		}

		if err = morseWorkspace.addAccount(accountAddr); err != nil {
			return err
		}

		coins := exportAccount.Coins

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
		if coin.Denom != pocket.DenomuPOKT {
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

		if !morseWorkspace.hasAccount(appAddr) {
			logger.Logger.Warn().
				Str("app_address", appAddr).
				Msg("no account found for application")

			// DEV_NOTE: If no auth account was found for this application, create a new one.
			if err := morseWorkspace.addAccount(appAddr); err != nil {
				return fmt.Errorf(
					"adding application account to account balance of address %q: %w",
					appAddr, err,
				)
			}
		}

		if exportApplication.StakedTokens != "" {
			if err := morseWorkspace.addAppStake(exportApplication); err != nil {
				return fmt.Errorf(
					"adding application stake amount to account balance of address %q: %w",
					appAddr, err,
				)
			}
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

		if !morseWorkspace.hasAccount(supplierAddr) {
			logger.Logger.Warn().
				Str("supplier_address", supplierAddr).
				Msg("no account found for supplier")

			// DEV_NOTE: If no auth account was found for this supplier, create a new one.
			if err := morseWorkspace.addAccount(supplierAddr); err != nil {
				return fmt.Errorf(
					"adding supplier account to account balance of address %q: %w",
					supplierAddr, err,
				)
			}
		}

		if exportSupplier.StakedTokens != "" {
			if err := morseWorkspace.addSupplierStake(exportSupplier); err != nil {
				return fmt.Errorf(
					"adding supplier stake amount to account balance of address %q: %w",
					supplierAddr, err,
				)
			}
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

// loadMorseState loads the MorseStateExport from the given path and deserializes it.
func loadMorseState(morseStateExportPath string) (*migrationtypes.MorseStateExport, error) {
	morseStateExportJSONBz, err := os.ReadFile(morseStateExportPath)
	if err != nil {
		return nil, err
	}

	morseStateExport := new(migrationtypes.MorseStateExport)
	if err = cmtjson.Unmarshal(morseStateExportJSONBz, morseStateExport); err != nil {
		return nil, err
	}

	return morseStateExport, nil
}
