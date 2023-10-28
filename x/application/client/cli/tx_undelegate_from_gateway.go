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

func CmdUndelegateFromGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegate-from-gateway [gateway address]",
		Short: "Undelegate an application from a gateway",
		Long: `Undelegate an application from the gateway whose address is provided. This is a broadcast operation
that will remove the gateway from those which the application
has delegated its trust to. This gateway will no longer have
the ability to sign relays from the application.

Example:
$ pocketd --home=$(POCKETD_HOME) tx application undelegate-from-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE)`,
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
