package migrate

import (
	"fmt"
	"io"
	"os"

	cosmosmath "cosmossdk.io/math"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const defaultLogOutput = "-"

var (
	flagDebugAccountsPerLog int
	flagLogLevel            string
	flagLogOutput           string
	logger                  polylog.Logger

	collectMorseAccountsCmd = &cobra.Command{
		Use:   "collect-morse-accounts [morse-state-export-path] [morse-account-state-path]",
		Args:  cobra.ExactArgs(2),
		Short: "Collect all account balances and corresponding stakes from the JSON file at [morse-state-export-path] and outputs them as JSON to [morse-account-state-path]",
		Long: `Collects the account balances and corresponding stakes from the MorseStateExport JSON file at morse-state-path
and outputs them as a MorseAccountState JSON to morse-accounts-path for use with
Shannon's MsgUploadMorseState. The Morse state export is generated via the Morse CLI:
pocket util export-genesis-for-reset [height] [new-chain-id] > morse-state-export.json`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var (
				logOutput io.Writer
				err       error
			)
			logLevel := polyzero.ParseLevel(flagLogLevel)
			if flagLogOutput == defaultLogOutput {
				logOutput = os.Stdout
			} else {
				logOutput, err = os.Open(flagLogOutput)
				if err != nil {
					return err
				}
			}

			logger = polyzero.NewLogger(
				polyzero.WithLevel(logLevel),
				polyzero.WithOutput(logOutput),
			).With("cmd", "migrate")
			return nil
		},
		RunE: runCollectMorseAccounts,
	}
)

func MigrateCmd() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migration commands",
	}
	migrateCmd.AddCommand(collectMorseAccountsCmd)
	migrateCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "The logging level (debug|info|warn|error)")
	migrateCmd.PersistentFlags().StringVar(&flagLogOutput, "log-output", defaultLogOutput, "The logging output (file path); defaults to stdout")

	collectMorseAccountsCmd.Flags().IntVar(&flagDebugAccountsPerLog, "debug-accounts-per-log", 0, "The number of accounts to log per debug message")

	return migrateCmd
}

// runCollectedMorseAccounts is run by the `poktrolld migrate collect-morse-accounts` command.
func runCollectMorseAccounts(_ *cobra.Command, args []string) error {
	// DEV_NOTE: No need to check args length due to cobra.ExactArgs(2).
	morseStateExportPath := args[0]
	morseAccountStatePath := args[1]

	logger.Info().
		Str("morse_state_export_path", morseStateExportPath).
		Str("morse_account_state_path", morseAccountStatePath).
		Msg("collecting Morse accounts...")

	morseWorkspace, err := collectMorseAccounts(morseStateExportPath, morseAccountStatePath)
	if err != nil {
		return err
	}

	return morseWorkspace.infoLogComplete()
}

// collectMorseAccounts reads and transforms the JSON serialized MorseStateExport
// at morseStateExportPath into a JSON serialized MorseAccountState, and then writes
// it to morseAccountStatePath.
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
func transformMorseState(
	inputState *migrationtypes.MorseStateExport,
	morseWorkspace *morseImportWorkspace,
) error {
	// Iterate over accounts and copy the balances.
	logger.Info().Msg("collecting account balances...")
	if err := collectInputAccountBalances(inputState, morseWorkspace); err != nil {
		return err
	}

	// Iterate over applications and add the stakes to the corresponding account balances.
	logger.Info().Msg("collecting application stakes...")
	if err := collectInputApplicationStakes(inputState, morseWorkspace); err != nil {
		return err
	}

	// Iterate over suppliers and add the stakes to the corresponding account balances.
	logger.Info().Msg("collecting supplier stakes...")
	return collectInputSupplierStakes(inputState, morseWorkspace)
}

// collectInputAccountBalances iterates over the accounts in the inputState and
// adds the balances to the corresponding account balances in the morseWorkspace.
func collectInputAccountBalances(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for exportAccountIdx, exportAccount := range inputState.AppState.Auth.Accounts {
		if shouldDebugLogProgress(exportAccountIdx) {
			morseWorkspace.debugLogProgress(exportAccountIdx)
		}

		// DEV_NOTE: Ignore module accounts.
		if exportAccount.Type != "posmint/Account" {
			logger.Warn().
				Str("type", exportAccount.Type).
				Str("address", exportAccount.Value.Address.String()).
				Str("coins", fmt.Sprintf("%s", exportAccount.Value.Coins)).
				Msg("ignoring non-EOA account")
			continue
		}

		accountAddr := exportAccount.Value.Address.String()
		if _, _, err := morseWorkspace.ensureAccount(accountAddr, exportAccount); err != nil {
			return err
		}

		coins := exportAccount.Value.Coins

		// If, for whatever reason, the account has no coins, skip it.
		// DEV_NOTE: This is NEVER expected to happen, but is technically possible.
		if len(coins) == 0 {
			logger.Warn().Str("address", accountAddr).Msg("account has no coins; skipping")
			return nil
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

		if err := morseWorkspace.addUpokt(accountAddr, coin.Amount); err != nil {
			return fmt.Errorf(
				"adding morse account balance (%s) to account balance of address %q: %w",
				coin, accountAddr, err,
			)
		}
	}
	return nil
}

// shouldDebugLogProgress returns true if the given exportAccountIdx should be logged
// via debugLogProgress.
func shouldDebugLogProgress(exportAccountIdx int) bool {
	return flagDebugAccountsPerLog > 0 &&
		exportAccountIdx%flagDebugAccountsPerLog == 0
}

// collectInputApplicationStakes iterates over the applications in the inputState and
// adds the stake to the corresponding account balances in the morseWorkspace.
func collectInputApplicationStakes(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for _, exportApplication := range inputState.AppState.Application.Applications {
		appAddr := exportApplication.Address.String()

		// DEV_NOTE: An account SHOULD exist for each actor.
		if !morseWorkspace.hasAccount(appAddr) {
			return ErrMorseExportState.Wrapf("account not found corresponding to application with address %q", appAddr)
		}

		appStakeAmtUpokt, ok := cosmosmath.NewIntFromString(exportApplication.StakedTokens)
		if !ok {
			return ErrMorseExportState.Wrapf("failed to parse application stake amount %q", exportApplication.StakedTokens)
		}

		if err := morseWorkspace.addUpokt(appAddr, appStakeAmtUpokt); err != nil {
			return fmt.Errorf(
				"adding application stake amount to account balance of address %q: %w",
				appAddr, err,
			)
		}
	}
	return nil
}

// collectInputSupplierStakes iterates over the suppliers in the inputState and
// adds the stake to the corresponding account balances in the morseWorkspace.
func collectInputSupplierStakes(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for _, exportSupplier := range inputState.AppState.Pos.Validators {
		supplierAddr := exportSupplier.Address.String()

		// DEV_NOTE: An account SHOULD exist for each actor.
		if !morseWorkspace.hasAccount(supplierAddr) {
			return ErrMorseExportState.Wrapf("account not found corresponding to supplier with address %q", supplierAddr)
		}

		supplierStakeAmtUpokt, ok := cosmosmath.NewIntFromString(exportSupplier.StakedTokens)
		if !ok {
			return ErrMorseExportState.Wrapf("failed to parse supplier stake amount %q", exportSupplier.StakedTokens)
		}

		if err := morseWorkspace.addUpokt(supplierAddr, supplierStakeAmtUpokt); err != nil {
			return fmt.Errorf(
				"adding supplier stake amount to account balance of address %q: %w",
				supplierAddr, err,
			)
		}
	}
	return nil
}
