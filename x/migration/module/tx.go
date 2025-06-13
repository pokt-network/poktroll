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
	migrationCmd := &cobra.Command{
		Use:   "migration",
		Short: "Transactions commands for the migration module",
	}

	// Global logger flags
	// DEV_NOTE: Since the root command runs logger.PreRunESetup(), we need to ensure
	// that the log level and output flags are registered on all migration module tx subcommands.
	// TODO_INVESTIGATE(@bryanchriswhite):
	// 1. Why isn't this a double registration
	// 2. Is this still explicitly necessary?
	// 3. What's different about other module TX (sub)commands?
	migrationCmd.PersistentFlags().StringVar(&logger.LogLevel, cosmosflags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	migrationCmd.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	// Register the --auto-fee flag for all migration module tx subcommands.
	migrationCmd.PersistentFlags().Bool(flags.FlagAutoFee, flags.DefaultFlagAutoFee, flags.FlagAutoFeeUsage)

	// Register all migration module tx subcommands.
	migrationCmd.AddCommand(cmd.CollectMorseAccountsCmd())
	migrationCmd.AddCommand(cmd.ClaimAccountCmd())
	migrationCmd.AddCommand(cmd.ClaimMorseAccountBulkCmd())
	migrationCmd.AddCommand(cmd.ClaimApplicationCmd())
	migrationCmd.AddCommand(cmd.ClaimSupplierCmd())
	migrationCmd.AddCommand(cmd.ClaimSupplierBulkCmd())
	migrationCmd.AddCommand(cmd.ImportMorseAccountsCmd())
	migrationCmd.AddCommand(cmd.ValidateMorseAccountsCmd())
	migrationCmd.AddCommand(cmd.RecoverMorseAccountCmd())

	return migrationCmd
}
