package cli

import (
	"strconv"

	"pocket/x/gateway/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeGateway() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "unstake-gateway [amount]",
		Short: "Unstake a gateway",
		Long: `Unstake a gateway. This is a broadcast operation that will unstake
the gateway specified by the 'from' address.
Example:
$ pocketd --home=$(POCKETD_HOME) tx gateway unstake-gateway --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := types.NewMsgUnstakeGateway(
				clientCtx.GetFromAddress().String(),
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
