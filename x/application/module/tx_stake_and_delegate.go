package application

import (
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/application/module/config"
	"github.com/pokt-network/poktroll/x/application/types"
)

func CmdStakeAndDelegateToGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-and-delegate --config <config_file.yaml>",
		Short: "Stake an application and delegate to one or more gateways in a single transaction",
		Long: `Stake an application and delegate to one or more gateways in a single transaction.
The config file extends the standard stake config with an optional 'gateway_addresses' field:

  stake_amount: "1000000upokt"
  service_ids:
    - "svc1"
  gateway_addresses:
    - "pokt1gateway1..."
    - "pokt1gateway2..."

Example:
$ pocketd tx application stake-and-delegate --config stake_config.yaml --keyring-backend test --from $(APP) --network=<network> --home $(POCKETD_HOME)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			configFile, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			configContent, err := os.ReadFile(configFile)
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

			appAddr := clientCtx.GetFromAddress().String()

			msgStake := types.NewMsgStakeApplication(
				appAddr,
				appStakeConfigs.StakeAmount,
				appStakeConfigs.Services,
			)
			if err := msgStake.ValidateBasic(); err != nil {
				return err
			}

			msgs := []cosmostypes.Msg{msgStake}

			for _, gwAddr := range appStakeConfigs.GatewayAddresses {
				msgDelegate := types.NewMsgDelegateToGateway(appAddr, gwAddr)
				if err := msgDelegate.ValidateBasic(); err != nil {
					return err
				}
				msgs = append(msgs, msgDelegate)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	cmd.Flags().String("config", "", "Path to the stake config file")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
