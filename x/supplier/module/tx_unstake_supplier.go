package supplier

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

func CmdUnstakeSupplier() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "unstake-supplier <operator_address>",
		Short: "Unstake a supplier",
		Long: `Unstake an supplier with the provided parameters. This is a broadcast operation that will unstake the supplier specified by the <operator_address> and owned by 'from' address.

Example:
$ poktrolld tx supplier unstake-supplier $(OPERATOR_ADDRESS) --keyring-backend test --from $(OWNER_ADDRESS) --node $(POCKET_NODE) --home=$(POKTROLLD_HOME)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// address is the operator address of the supplier
			address := args[0]

			msg := types.NewMsgUnstakeSupplier(
				clientCtx.GetFromAddress().String(),
				address,
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
