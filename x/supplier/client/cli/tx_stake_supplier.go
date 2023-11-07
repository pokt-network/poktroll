package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

var _ = strconv.Itoa(0)

func CmdStakeSupplier() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		// TODO_HACK: For now we are only specifying the service IDs as a list of of strings separated by commas.
		// This needs to be expand to specify the full SupplierServiceConfig. Furthermore, providing a flag to
		// a file where SupplierServiceConfig specifying full service configurations in the CLI by providing a flag that accepts a JSON string
		Use:   "stake-supplier [amount] [svcId1;url1,svcId2;url2,...,svcIdN;urlN]",
		Short: "Stake a supplier",
		Long: `Stake an supplier with the provided parameters. This is a broadcast operation that
will stake the tokens and associate them with the supplier specified by the 'from' address.
TODO_HACK: Until proper service configuration files are supported, suppliers must specify the services as a single string
of comma separated values of the form 'service;url' where 'service' is the service ID and 'url' is the service URL.
For example, an application that stakes for 'anvil' could be matched with a supplier staking for 'anvil;http://anvil:8547'.

Example:
$ poktrolld --home=$(POCKETD_HOME) tx supplier stake-supplier 1000upokt anvil;http://anvil:8547 --keyring-backend test --from $(APP) --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			stakeString := args[0]
			servicesArg := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			stake, err := sdk.ParseCoinNormalized(stakeString)
			if err != nil {
				return err
			}

			services, err := hackStringToServices(servicesArg)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeSupplier(
				clientCtx.GetFromAddress().String(),
				stake,
				services,
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

// TODO_BLOCKER, TODO_HACK: The supplier stake command should take an argument
// or flag that points to a file containing all the services configurations & specifications.
// As a quick workaround, we just need the service & url to get things working for now.
func hackStringToServices(servicesArg string) ([]*sharedtypes.SupplierServiceConfig, error) {
	supplierServiceConfig := make([]*sharedtypes.SupplierServiceConfig, 0)
	serviceStrings := strings.Split(servicesArg, ",")
	for _, serviceString := range serviceStrings {
		serviceParts := strings.Split(serviceString, ";")
		if len(serviceParts) != 2 {
			return nil, fmt.Errorf("invalid service string: %s. Expected it to be of the form 'service;url'", serviceString)
		}
		service := &sharedtypes.SupplierServiceConfig{
			ServiceId: &sharedtypes.ServiceId{
				Id: serviceParts[0],
			},
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     serviceParts[1],
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: make([]*sharedtypes.ConfigOption, 0),
				},
			},
		}
		supplierServiceConfig = append(supplierServiceConfig, service)
	}
	return supplierServiceConfig, nil
}
