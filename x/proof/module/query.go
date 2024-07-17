package proof

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/pokt-network/poktroll/x/proof/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the cli query commands for this module
// TODO_TECHDEBT(@bryanchriswhite, #370): remove if custom query commands are consolidated into AutoCLI.
func (am AppModule) GetQueryCmd() *cobra.Command {
	// Group supplier queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListClaims())
	cmd.AddCommand(CmdShowClaim())
	cmd.AddCommand(CmdListProof())
	cmd.AddCommand(CmdShowProof())
	// this line is used by starport scaffolding # 1

	return cmd
}
