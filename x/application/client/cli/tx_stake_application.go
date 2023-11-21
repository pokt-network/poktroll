package cli

import (
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/application/client/config"
	"github.com/pokt-network/poktroll/x/application/types"
)

var (
	flagStakeConfig string
	_               = strconv.Itoa(0)
)

func CmdStakeApplication() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		// This needs to be expand to specify the full ApplicationServiceConfig. Furthermore, providing a flag to
		// a file where ApplicationServiceConfig specifying full service configurations in the CLI by providing a flag that accepts a JSON string
		Use:   "stake-application <upokt_amount> --config <config_file.yaml>",
		Short: "Stake an application",
		Long: `Stake an application with the provided parameters. This is a broadcast operation that
will stake the tokens and serviceIds and associate them with the application specified by the 'from' address.

Example:
$ poktrolld --home=$(POKTROLLD_HOME) tx application stake-application 1000upokt --config stake_config.yaml --keyring-backend test --from $(APP) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			stakeString := args[0]
			configContent, err := os.ReadFile(flagStakeConfig)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			stake, err := sdk.ParseCoinNormalized(stakeString)
			if err != nil {
				return err
			}

			appStakeConfigs, err := config.ParseApplicationConfigs(configContent)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeApplication(
				clientCtx.GetFromAddress().String(),
				stake,
				appStakeConfigs,
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
