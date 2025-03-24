package gateway

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/pokt-network/pocket/x/gateway/types"
)

// GetQueryCmd returns the cli query commands for this module
// TODO_MAINNET(#370): remove if custom query commands are consolidated into AutoCLI.
func (am AppModule) GetQueryCmd() *cobra.Command {
	// Group gateway queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListGateway())
	cmd.AddCommand(CmdShowGateway())
	// this line is used by starport scaffolding # 1

	return cmd
}
