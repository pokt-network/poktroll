package application

import (
	"strconv"

	"github.com/pokt-network/pocket/x/application/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUndelegateFromGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegate-from-gateway [gateway address]",
		Short: "Undelegate an application from a gateway",
		Long: `Undelegate an application from the gateway with the provided address. This is a broadcast operation
that removes the authority from the gateway specified to sign relays requests for the application, disallowing the gateway
act on the behalf of the application during a session.

Example:
$ pocketd tx application undelegate-from-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			gatewayAddress := args[0]
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUndelegateFromGateway(
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
