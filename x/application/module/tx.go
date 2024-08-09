package application

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/application/types"
)

// GetTxCmd returns the transaction commands for this module
// TODO_MAINNET(#370): remove if custom query commands are consolidated into AutoCLI.
func (am AppModule) GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdStakeApplication())
	cmd.AddCommand(CmdUnstakeApplication())

	cmd.AddCommand(CmdDelegateToGateway())
	cmd.AddCommand(CmdUndelegateFromGateway())
	// this line is used by starport scaffolding # 1

	return cmd
}
