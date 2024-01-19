package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/service/types"
)

var _ = strconv.Itoa(0)

func CmdAddService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-service <service_id> <service_name>",
		Short: "Add a new service to the network",
		Long: `Add a new service to the network that will be available for applications,
gateways and suppliers to use. The service id MUST be unique - or the command
will fail, however the name you use to describe it does not have to be unique.

Example:
$ poktrolld tx service add-service "srv1" "service_one" --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE) --home=$(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			serviceIdStr := args[0]
			serviceNameStr := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddService(
				clientCtx.GetFromAddress().String(),
				serviceIdStr,
				serviceNameStr,
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
