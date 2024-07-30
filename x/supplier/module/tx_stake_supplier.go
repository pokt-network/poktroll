package supplier

import (
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/supplier/config"
	"github.com/pokt-network/poktroll/x/supplier/types"
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
		Long: `Stake an supplier with the provided parameters. This is a broadcast operation that
will stake the tokens and associate them with the supplier specified by the 'from' address.

Example:
$ poktrolld tx supplier stake-supplier --config stake_config.yaml --keyring-backend test --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,

		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			configContent, err := os.ReadFile(flagStakeConfig)
			if err != nil {
				return err
			}

			supplierStakeConfigs, err := config.ParseSupplierConfigs(configContent)
			if err != nil {
				return err
			}

			// Ensure the --from flag is set before getting the client context.
			if cmd.Flag(flags.FlagFrom) == nil {
				cmd.Flags().Set(flags.FlagFrom, supplierStakeConfigs.OwnerAddress)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Ensure the from address is the same as the owner address in the stake config file.
			if clientCtx.GetFromAddress().String() != supplierStakeConfigs.OwnerAddress {
				return types.ErrSupplierInvalidAddress.Wrapf(
					"operator address %q in the stake config file does not match the operator address %q in the message",
					supplierStakeConfigs.OperatorAddress,
					clientCtx.GetFromAddress().String(),
				)
			}

			msg := types.NewMsgStakeSupplier(
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
