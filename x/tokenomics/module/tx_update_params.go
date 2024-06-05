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

// TODO_BLOCKER(@bryanchriswhite, #322): Update the CLI once we determine settle on how to maintain and update parameters.
func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params <compute_units_to_tokens_multiplier>",
		Short: "Update the parameters of the tokenomics module",
		Long: `Update the parameters in the tokenomics module.",

All parameters must be provided when updating.

Example:
$ poktrolld tx tokenomics update-params <compute_units_to_tokens_multiplier> --from pnf --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Parse computeUnitsToTokensMultiplier
			computeUnitsToTokensMultiplier, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			// Get client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Create update params message
			msg := types.NewMsgUpdateParams(
				clientCtx.GetFromAddress().String(),
				computeUnitsToTokensMultiplier,
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
