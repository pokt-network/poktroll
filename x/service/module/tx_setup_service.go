package service

// TODO_MAINNET(@red-0ne): Add `UpdateService` or modify `AddService` to `UpsertService` to allow service owners
// to update parameters of existing services. This will requiring updating `proto/pocket/service/tx.proto` and
// all downstream code paths.
import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ = strconv.Itoa(0)

func CmdSetupService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("setup-service <service_id> <service_name> [compute_units_per_relay: default={%d}] [service_owner: default={--from address}]", types.DefaultComputeUnitsPerRelay),
		Short: "Add a new service or update an existing one in the network",
		Long: `Add a new service or update an existing one in the network that will be available for applications,
gateways and suppliers to use. The service id MUST be unique but the service name doesn't have to be.

Example:
$ pocketd tx service setup-service "svc1" "service_one" 1 $(SERVICE_OWNER) --keyring-backend test --from $(SIGNER) --network=<network> --home $(POCKETD_HOME)`,
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
				computeUnitsPerRelayArg, err := strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return sharedtypes.ErrSharedInvalidComputeUnitsPerRelay.Wrapf("unable to parse as uint64: %s", args[2])
				}
				if computeUnitsPerRelayArg == 0 {
					fmt.Printf("Using default compute_units_per_relay: %d\n", types.DefaultComputeUnitsPerRelay)
				}
			} else {
				fmt.Printf("Using default compute_units_per_relay: %d\n", types.DefaultComputeUnitsPerRelay)
			}

			serviceOwnerAddress := clientCtx.GetFromAddress().String()
			// Update the service owner address if the argument is provided.
			// Otherwise use the transaction signer address from the client context
			if len(args) > 3 {
				serviceOwnerAddress = args[3]
			} else {
				fmt.Printf("Using default service owner address: %s\n", serviceOwnerAddress)
			}

			msg := types.NewMsgSetupService(
				clientCtx.GetFromAddress().String(),
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
