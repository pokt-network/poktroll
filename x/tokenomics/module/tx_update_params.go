package tokenomics

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

var _ = strconv.Itoa(0)

// TODO_UPNEXT(@bryanchriswhite): Remove this. It's not used nor useful.
// Parameter updates currently happen via authz exec messages and in the
// future will be committed via governance proposals.
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params",
		Short: "Update the parameters of the tokenomics module",
		Long: `Update the parameters in the tokenomics module.",

All parameters must be provided when updating.

Example:
$ poktrolld tx tokenomics update-params --from pnf --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Get client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Create update params message
			msg := types.NewMsgUpdateParams(
				clientCtx.GetFromAddress().String(),
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			res := tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
			return res
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
