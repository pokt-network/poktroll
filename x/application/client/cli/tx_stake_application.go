package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"pocket/x/application/types"
)

var _ = strconv.Itoa(0)

func CmdStakeApplication() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "stake-application [amount]",
		Short: "Stake an application",
		Long: `Stake an application with the provided parameters. This is a broadcast operation that
will stake the tokens and associate them with the application specified by the 'from' address.

Example:
$ pocketd --home=$(POCKETD_HOME) tx application stake-application 1000upokt --keyring-backend test --from $(APP) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			stakeString := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			stake, err := sdk.ParseCoinNormalized(stakeString)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeApplication(
				clientCtx.GetFromAddress().String(),
				stake,
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
