package cli

import (
	"strconv"

	"pocket/x/application/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdDelegateToGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate-to-gateway [gateway address]",
		Short: "Delegate an application to a gateway",
		Long: `Delegate an application to the gateway with the provided address. This is a broadcast operation
that delegates authority to the gateway specified to sign relays requests for the application, allowing the gateway
act on the behalf of the application during a session.

Example:
$ pocketd --home=$(POCKETD_HOME) tx application delegate-to-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			gatewayAddress := args[0]
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDelegateToGateway(
				clientCtx.GetFromAddress().String(),
				gatewayAddress,
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
