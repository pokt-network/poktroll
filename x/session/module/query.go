package session

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/pokt-network/pocket/x/session/types"
)

// GetQueryCmd returns the cli query commands for this module
func (am AppModule) GetQueryCmd() *cobra.Command {
	// Group session queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdGetSession())

	// this line is used by starport scaffolding # 1

	return cmd
}
