package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

var _ = strconv.Itoa(0)

func CmdStakeGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-gateway <upokt_amount>",
		Short: "Stake a gateway",
		Long: `Stake a gateway with the provided parameters. This is a broadcast operation that
will stake the tokens and associate them with the gateway specified by the 'from' address.
Example:
$ poktrolld --home=$(POKTROLLD_HOME) tx gateway stake-gateway 1000upokt --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			stakeString := args[0]
			stake, err := sdk.ParseCoinNormalized(stakeString)
			if err != nil {
				return err
			}
			msg := types.NewMsgStakeGateway(
				clientCtx.GetFromAddress().String(),
				stake,
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
