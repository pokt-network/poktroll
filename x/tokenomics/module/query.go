package tokenomics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetQueryCmd returns the cli query commands for this module
// TODO_TECHDEBT(#370): remove if custom query commands are consolidated into AutoCLI.
func (am AppModule) GetQueryCmd() *cobra.Command {
	// Group tokenomics queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	// this line is used by starport scaffolding # 1

	return cmd
}
