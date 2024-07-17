package application

import (
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/x/application/module/config"
)

var (
	flagStakeConfig string
	_               = strconv.Itoa(0)
)

func CmdStakeApplication() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "stake-application --config <config_file.yaml>",
		Short: "Stake an application",
		Long: `Stake an application with the provided parameters. This is a broadcast operation that
will stake the tokens and serviceIds and associate them with the application specified by the 'from' address.

Example:
$ poktrolld tx application stake-application --config stake_config.yaml --keyring-backend test --from $(APP) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			configContent, err := os.ReadFile(flagStakeConfig)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			appStakeConfigs, err := config.ParseApplicationConfigs(configContent)
			if err != nil {
				return err
			}

			msg := application.NewMsgStakeApplication(
				clientCtx.GetFromAddress().String(),
				appStakeConfigs.StakeAmount,
				appStakeConfigs.Services,
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
