package migrate

import (
	"fmt"
	"os"

	cosmosmath "cosmossdk.io/math"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

var collectMorseAccountsCmd = &cobra.Command{
	Use:   "collect-morse-accounts [morse-state-path] [morse-accounts-path]",
	Args:  cobra.ExactArgs(2),
	Short: "Collect all account balances and corresponding stakes from the JSON file at [morse-state-path] and outputs them as JSON to [morse-accounts-path]",
	Long: `Collects the account balances and corresponding stakes from the MorseStateExport JSON file at morse-state-path
and outputs them as a MorseAccountState JSON to morse-accounts-path for use with
Shannon's MsgUploadMorseState. The Morse state export is generated via the Morse CLI:
pocket util export-genesis-for-reset [height] [new-chain-id] > morse-state-export.json`,
	RunE: runCollectMorseAccounts,
}

func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migration commands",
	}
	cmd.AddCommand(collectMorseAccountsCmd)

	return cmd
}

// runCollectedMorseAccounts is run by the `poktrolld migrate collect-morse-accounts` command.
func runCollectMorseAccounts(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	outputPath := args[1]

	return collectMorseAccounts(inputPath, outputPath)
}

// collectMorseAccounts transforms the JSON serialized MorseStateExport at
// inputStatePath into a JSON serialized MorseAccountState at outputStatePath.
func collectMorseAccounts(inputStatePath, outputStatePath string) error {
	if err := validatePathIsFile(inputStatePath); err != nil {
		return err
	}

	inputStateJSON, err := os.ReadFile(inputStatePath)
	if err != nil {
		return err
	}

	inputState := new(migrationtypes.MorseStateExport)
	if err = cmtjson.Unmarshal(inputStateJSON, inputState); err != nil {
		return err
	}

	outputStateJSON, err := transformMorseState(inputState)
	if err != nil {
		return err
	}

	if err = os.WriteFile(outputStatePath, outputStateJSON, 0644); err != nil {
		return err
	}

	return nil
}

// validatePathIsFile returns an error if the given path does not exist or is not a file.
func validatePathIsFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("[morse-JSON-input-path] cannot be a directory")
	}

	return nil
}

// transformMorseState consolidates the Morse account balance, application stake,
// and supplier stake for each account as an entry in the resulting MorseAccountState.
func transformMorseState(inputState *migrationtypes.MorseStateExport) ([]byte, error) {
	morseWorkspace := &morseImportWorkspace{
		addressToIdx: make(map[string]uint64),
		accounts:     make([]*migrationtypes.MorseAccount, 0),
	}

	// Iterate over accounts and copy the balances.
	if err := collectInputAccountBalances(inputState, morseWorkspace); err != nil {
		return nil, err
	}

	// Iterate over applications and add the stakes to the corresponding account balances.
	if err := collectInputApplicationStakes(inputState, morseWorkspace); err != nil {
		return nil, err
	}

	// Iterate over suppliers and add the stakes to the corresponding account balances.
	err := collectInputSupplierStakes(inputState, morseWorkspace)
	if err != nil {
		return nil, err
	}

	morseAccountState := &migrationtypes.MorseAccountState{Accounts: morseWorkspace.accounts}
	return cmtjson.Marshal(morseAccountState)
}

// collectInputAccountBalances iterates over the accounts in the inputState and
// adds the balances to the corresponding account balances in the morseWorkspace.
func collectInputAccountBalances(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for _, exportAccount := range inputState.AppState.Auth.Accounts {
		// DEV_NOTE: Ignore module accounts.
		if exportAccount.Type != "posmint/Account" {
			continue
		}

		addr := exportAccount.Value.Address.String()
		morseWorkspace.ensureAccount(addr, exportAccount)

		coins := exportAccount.Value.Coins
		if len(coins) == 0 {
			return nil
		}

		// DEV_NOTE: SHOULD ONLY be one denom (upokt).
		coin := coins[0]
		if coin.Denom != volatile.DenomuPOKT {
			return fmt.Errorf("unsupported denom %q", coin.Denom)
		}

		if err := morseWorkspace.addUpokt(addr, coin.Amount); err != nil {
			return err
		}
	}
	return nil
}

// collectInputApplicationStakes iterates over the applications in the inputState and
// adds the stake to the corresponding account balances in the morseWorkspace.
func collectInputApplicationStakes(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for _, exportApplication := range inputState.AppState.Application.Applications {
		addr := exportApplication.Address.String()

		// DEV_NOTE: An account SHOULD exist for each actor.
		if !morseWorkspace.hasAccount(addr) {
			// TODO_IN_THIS_COMMIT: consolidate error types...
			return fmt.Errorf("account %q not found", addr)
		}

		appStakeAmtUpokt, ok := cosmosmath.NewIntFromString(exportApplication.StakedTokens)
		if !ok {
			return fmt.Errorf("failed to parse application stake amount %q", exportApplication.StakedTokens)
		}

		if err := morseWorkspace.addUpokt(addr, appStakeAmtUpokt); err != nil {
			return err
		}
	}
	return nil
}

// collectInputSupplierStakes iterates over the suppliers in the inputState and
// adds the stake to the corresponding account balances in the morseWorkspace.
func collectInputSupplierStakes(inputState *migrationtypes.MorseStateExport, morseWorkspace *morseImportWorkspace) error {
	for _, exportSupplier := range inputState.AppState.Pos.Validators {
		addr := exportSupplier.Address.String()

		// DEV_NOTE: An account SHOULD exist for each actor.
		if !morseWorkspace.hasAccount(addr) {
			// TODO_IN_THIS_COMMIT: consolidate error types...
			return fmt.Errorf("account %q not found", addr)
		}

		supplierStakeAmtUpokt, ok := cosmosmath.NewIntFromString(exportSupplier.StakedTokens)
		if !ok {
			return fmt.Errorf("failed to parse supplier stake amount %q", exportSupplier.StakedTokens)
		}

		if err := morseWorkspace.addUpokt(addr, supplierStakeAmtUpokt); err != nil {
			return err
		}
	}
	return nil
}
