package cmd

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/core/appmodule"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
)

// TODO_IN_THIS_COMMIT: update comment... copied from cosmossdk.io/api/cosmos/autocli/util.go
// TopLevelCmd creates a new top-level command with the provided name and
// description. The command will have DisableFlagParsing set to false and
// SuggestionsMinimumDistance set to 2.
func TopLevelCmd(ctx context.Context, use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        use,
		Short:                      short,
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       validateCmd,
	}
	cmd.SetContext(ctx)

	return cmd
}

// TODO_IN_THIS_COMMIT: update comment... copied from cosmossdk.io/api/cosmos/autocli/validate.go
// validateCmd returns unknown command error or Help display if help flag set
func validateCmd(cmd *cobra.Command, args []string) error {
	var unknownCmd string
	var skipNext bool

	for _, arg := range args {
		// search for help flag
		if arg == "--help" || arg == "-h" {
			return cmd.Help()
		}

		// check if the current arg is a flag
		switch {
		case len(arg) > 0 && (arg[0] == '-'):
			// the next arg should be skipped if the current arg is a
			// flag and does not use "=" to assign the flag's value
			if !strings.Contains(arg, "=") {
				skipNext = true
			} else {
				skipNext = false
			}
		case skipNext:
			// skip current arg
			skipNext = false
		case unknownCmd == "":
			// unknown command found
			// continue searching for help flag
			unknownCmd = arg
		}
	}

	// return the help screen if no unknown command is found
	if unknownCmd != "" {
		err := fmt.Sprintf("unknown command \"%s\" for \"%s\"", unknownCmd, cmd.CalledAs())

		// build suggestions for unknown argument
		if suggestions := cmd.SuggestionsFor(unknownCmd); len(suggestions) > 0 {
			err += "\n\nDid you mean this?\n"
			for _, s := range suggestions {
				err += fmt.Sprintf("\t%v\n", s)
			}
		}
		return fmt.Errorf(err)
	}

	return cmd.Help()
}

// TODO_IN_THIS_COMMIT: godoc & move...
type autoCLIAppModule interface {
	appmodule.AppModule
	AutoCLIOptions() *autocliv1.ModuleOptions
}

// TODO_IN_THIS_COMMIT: godoc & move...
func AddModuleAutoCLICommands(appModule appmodule.AppModule, moduleCmd *cobra.Command) {
	appAutoCLIModule, ok := appModule.(autoCLIAppModule)
	// If the module doesn't implement AutoCLIOptions(), this is a no-op.
	if !ok {
		return
	}

	// TODO_IN_THIS_COMMIT: comment... no need to create query commands, autoCLI will do that - they don't require modification.
	autoCLIOpts := appAutoCLIModule.AutoCLIOptions()
	// If the module has empty AutoCLI options, this is a no-op.
	if autoCLIOpts == nil {
		return
	}

	// If the module's AutoCLI options have no Tx options, this is a no-op.
	if autoCLIOpts.Tx == nil {
		return
	}

	// Add customized commands for each skipped (and uncommented) RPC method.
	for _, rpcCmdOpt := range autoCLIOpts.Tx.RpcCommandOptions {
		// TODO_IN_THIS_COMMIT: update comment... this is analogous to what autoCLI does internally.
		rpcMethodCmd := TopLevelCmd(moduleCmd.Context(), rpcCmdOpt.Use, rpcCmdOpt.Short)

		// Register the --auto-fee flag for the rpc method command.
		rpcMethodCmd.Flags().Bool(flags.FlagAutoFee, flags.DefaultFlagAutoFee, flags.FlagAutoFeeUsage)

		moduleCmd.AddCommand(rpcMethodCmd)
	}
}
