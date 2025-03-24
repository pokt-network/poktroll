package gateway

import (
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/gateway/module/config"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

var (
	flagStakeConfig string
	_               = strconv.Itoa(0)
)

func CmdStakeGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-gateway --config <config_file.yaml>",
		Short: "Stake a gateway",
		Long: `Stake a gateway with the provided parameters. This is a broadcast operation that
will stake the tokens and associate them with the gateway specified by the 'from' address.
Example:
$ pocketd tx gateway stake-gateway --config stake_config.yaml --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE) --home $(POCKETD_HOME)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			configContent, err := os.ReadFile(flagStakeConfig)
			if err != nil {
				return err
			}

			gatewayStakeConfig, err := config.ParseGatewayConfig(configContent)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeGateway(
				clientCtx.GetFromAddress().String(),
				gatewayStakeConfig.StakeAmount,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringVar(&flagStakeConfig, "config", "", "Path to the stake config file")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
