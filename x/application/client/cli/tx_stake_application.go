package cli

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/application/types"
)

var _ = strconv.Itoa(0)

func CmdStakeApplication() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		// TODO_HACK: For now we are only specifying the service IDs as a list of of strings separated by commas.
		// This needs to be expand to specify the full ApplicationServiceConfig. Furthermore, providing a flag to
		// a file where ApplicationServiceConfig specifying full service configurations in the CLI by providing a flag that accepts a JSON string
		Use:   "stake-application [amount] [svcId1,svcId2,...,svcIdN]",
		Short: "Stake an application",
		Long: `Stake an application with the provided parameters. This is a broadcast operation that
will stake the tokens and serviceIds and associate them with the application specified by the 'from' address.

Example:
$ poktrolld --home=$(POKTROLLD_HOME) tx application stake-application 1000upokt svc1,svc2,svc3 --keyring-backend test --from $(APP) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			stakeString := args[0]
			serviceIdsString := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			stake, err := sdk.ParseCoinNormalized(stakeString)
			if err != nil {
				return err
			}

			serviceIds := strings.Split(serviceIdsString, ",")

			msg := types.NewMsgStakeApplication(
				clientCtx.GetFromAddress().String(),
				stake,
				serviceIds,
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
