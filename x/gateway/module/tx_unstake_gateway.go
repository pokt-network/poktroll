package gateway

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	pocketflags "github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

var _ = strconv.Itoa(0)

func CmdUnstakeGateway() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "unstake-gateway <upokt_amount>",
		Short: "Unstake a gateway",
		Long: `Unstake a gateway. This is a broadcast operation that will unstake the gateway specified by the 'from' address.

Example:
$ pocketd tx gateway unstake-gateway --keyring-backend test --from $(GATEWAY) --network=<network> --home $(POCKETD_HOME)`,
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

	pocketflags.AddTxFlagsToCmd(cmd)

	return cmd
}
