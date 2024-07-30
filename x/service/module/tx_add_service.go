package service

// TODO_BETA: Add `UpdateService` or modify `AddService` to `UpsertService` to allow service owners
// to update parameters of existing services. This will requiring updating `proto/poktroll/service/tx.proto` and
// all downstream code paths.
import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/service/types"
)

var _ = strconv.Itoa(0)

// TODO_UPNEXT: Change `add-service` to `update-service` so the source owner can
// update the compute units per relay for an existing service. Make it possible
// to update a service (e.g. update # of compute units per relay). This will require
// search for all variations of `AddService` in the codebase (filenames, helpers, etc...),
// ensuring that only the owner can update it on chain, and tackling some of the tests in `service.feature`.
func CmdAddService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("add-service <service_id> <service_name> [compute_units_per_relay: default={%d}]", types.DefaultComputeUnitsPerRelay),
		Short: "Add a new service to the network",
		Long: `Add a new service to the network that will be available for applications,
gateways and suppliers to use. The service id MUST be unique but the service name doesn't have to be.

Example:
$ poktrolld tx service add-service "svc1" "service_one" 1 --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			serviceIdStr := args[0]
			serviceNameStr := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			computeUnitsPerRelay := types.DefaultComputeUnitsPerRelay
			// if compute units per relay argument is provided
			if len(args) > 2 {
				computeUnitsPerRelay, err = strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return types.ErrServiceInvalidComputeUnitsPerRelay.Wrapf("unable to parse as uint64: %s", args[2])
				}
			} else {
				fmt.Printf("Using default compute_units_per_relay: %d\n", types.DefaultComputeUnitsPerRelay)
			}

			serviceOwnerAddress := clientCtx.GetFromAddress().String()
			msg := types.NewMsgAddService(
				serviceOwnerAddress,
				serviceIdStr,
				serviceNameStr,
				computeUnitsPerRelay,
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
