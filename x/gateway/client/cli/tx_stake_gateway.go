package cli

import (
	"strconv"

	"pocket/x/gateway/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdStakeGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-gateway [amount]",
		Short: "Broadcast message stake-gateway",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			stakeAmountString := args[0]
			stakeAmount, err := sdk.ParseCoinNormalized(stakeAmountString)
			if err != nil {
				return err
			}
			msg := types.NewMsgStakeGateway(
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
