package cli

import (
	"strconv"

	"pocket/x/application/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStakeApplication() *cobra.Command {
	// fromAddress & signature is retrieved via flags.FlagFrom in the clientCtx
	cmd := &cobra.Command{
		Use:   "stake-application",
		Short: "Broadcast message stake-application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			stakeAmountString := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			stakeAmount, err := sdk.ParseCoinNormalized(stakeAmountString)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeApplication(
				clientCtx.GetFromAddress().String(),
				stakeAmount,
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
