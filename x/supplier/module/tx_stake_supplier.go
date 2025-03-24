package supplier

import (
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/pocket/x/supplier/config"
	"github.com/pokt-network/pocket/x/supplier/types"
)

var (
	flagStakeConfig string
	_               = strconv.Itoa(0) // Part of the default ignite imports
)

func CmdStakeSupplier() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "stake-supplier --config <config_file.yaml>",
		Short: "Stake a supplier",
		Long: `Stake a supplier using the specified configuration file. This command
supports both custodial and non-custodial staking of the signer's tokens.
It sources the necessary information from the provided configuration file.

For more details on the staking process, please refer to the supplier staking documentation at:
https://dev.pocket.com/operate/configs/supplier_staking_config

Example:
$ pocketd tx supplier stake-supplier --config stake_config.yaml --keyring-backend test  --from $(OWNER_ADDRESS) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,

		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			configContent, err := os.ReadFile(flagStakeConfig)
			if err != nil {
				return err
			}

			supplierStakeConfigs, err := config.ParseSupplierConfigs(cmd.Context(), configContent)
			if err != nil {
				return err
			}

			// Ensure the --from flag is set before getting the client context.
			if cmd.Flag(flags.FlagFrom) == nil {
				if err = cmd.Flags().Set(flags.FlagFrom, supplierStakeConfigs.OwnerAddress); err != nil {
					return err
				}
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeSupplier(
				clientCtx.GetFromAddress().String(),
				supplierStakeConfigs.OwnerAddress,
				supplierStakeConfigs.OperatorAddress,
				supplierStakeConfigs.StakeAmount,
				supplierStakeConfigs.Services,
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
