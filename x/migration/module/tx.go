package migration

import (
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/module/cmd"
)

// GetTxCmd returns the Cobra command corresponding to the migration module's
// tx subcommands (i.e. `poktrolld tx migration`).
//
// By implementing this method, NONE of the migration module's tx subcommands are
// generated automatically (i.e. via autoCLI).
// Instead, they are constructed here.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return TxCommands()
}

// TxCommands returns the Cobra command corresponding to migration module's tx
// subcommands (i.e. `poktrolld tx migration`).
//
// Since autoCLI does not apply to several migration CLI operations, this command
// MUST be manually constructed.
func TxCommands() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migration",
		Short: "Transactions commands for the migration module",
	}

	migrateCmd.AddCommand(cmd.CollectMorseAccountsCmd())
	migrateCmd.AddCommand(cmd.ClaimAccountCmd())
	migrateCmd.AddCommand(cmd.ClaimApplicationCmd())
	migrateCmd.AddCommand(cmd.ClaimSupplierCmd())
	migrateCmd.AddCommand(cmd.ImportMorseAccountsCmd())
	migrateCmd.PersistentFlags().StringVar(&logger.LogLevel, flags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	migrateCmd.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	return migrateCmd
}
