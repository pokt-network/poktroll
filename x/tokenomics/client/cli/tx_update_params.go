package cli

import (
	"log"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params <compute_units_to_tokens_multiplier>",
		Short: "Update the parameters of the tokenomics module",
		Long: `Update the parameter in the tokenomics module.",

All parameters must be provided when updating.

Example:
$ poktrolld tx tokenomics update-params <compute_units_to_tokens_multiplier> --from dao --node $(POCKET_NODE) --home=$(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Parse computeUnitsToTokensMultiplier
			computeUnitsToTokensMultiplier, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				log.Fatal(err)
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

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
