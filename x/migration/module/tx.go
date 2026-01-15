package migration

import (
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/module/cmd"
)

// GetTxCmd returns the Cobra command corresponding to the migration module's
// tx subcommands (i.e. `pocketd tx migration`).
//
// By implementing this method, NONE of the migration module's tx subcommands are
// generated automatically (i.e. via autoCLI).
// Instead, they are constructed here.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return TxCommands()
}

// TxCommands returns the Cobra command corresponding to migration module's tx
// subcommands (i.e. `pocketd tx migration`).
//
// Since autoCLI does not apply to several migration CLI operations, this command
// MUST be manually constructed.
func TxCommands() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migration",
		Short: "Transactions commands for the migration module",
	}

	// TODO_MAINNET_MIGRATION(@bryanchriswhite): Add `recover-morse-account` migration module tx command.
	// Be sure to include comprehensive `Long` and `Example` cobra.Command fields!

	migrateCmd.AddCommand(cmd.CollectMorseAccountsCmd())
	migrateCmd.AddCommand(cmd.ClaimAccountCmd())
	migrateCmd.AddCommand(cmd.ClaimMorseAccountBulkCmd())
	migrateCmd.AddCommand(cmd.ClaimApplicationCmd())
	migrateCmd.AddCommand(cmd.ClaimSupplierCmd())
	migrateCmd.AddCommand(cmd.ClaimSupplierBulkCmd())
	migrateCmd.AddCommand(cmd.ImportMorseAccountsCmd())
	migrateCmd.AddCommand(cmd.ValidateMorseAccountsCmd())
	migrateCmd.AddCommand(cmd.RecoverMorseAccountCmd())
	migrateCmd.AddCommand(cmd.AdminRecoverMorseAccountCmd())
	migrateCmd.PersistentFlags().StringVar(&logger.LogLevel, cosmosflags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	migrateCmd.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	return migrateCmd
}
