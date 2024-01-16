package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// CmdUnstakeSupplier creates a new unstake-supplier command.
func CmdUnstakeSupplier() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "unstake-supplier",
		Short: "Unstake a supplier",
		// Not liniting as this is the long description of the command.
		//nolint:lll
		Long: `Unstake an supplier with the provided parameters. This is a broadcast operation that will unstake the supplier specified by the 'from' address.

Example:
		$ poktrolld --home=$(POKTROLLD_HOME) tx supplier unstake-supplier --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUnstakeSupplier(
				clientCtx.GetFromAddress().String(),
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
