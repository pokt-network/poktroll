package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group supplier queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListSupplier())
	cmd.AddCommand(CmdShowSupplier())
	cmd.AddCommand(CmdListClaims())
	cmd.AddCommand(CmdShowClaim())
	cmd.AddCommand(CmdListProof())
	cmd.AddCommand(CmdShowProof())
	// this line is used by starport scaffolding # 1

	return cmd
}
