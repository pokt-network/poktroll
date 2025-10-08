package service

// TODO_MAINNET(@red-0ne): Add `UpdateService` or modify `AddService` to `UpsertService` to allow service owners
// to update parameters of existing services. This will requiring updating `proto/pocket/service/tx.proto` and
// all downstream code paths.
import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ = strconv.Itoa(0)

// TODO_POST_MAINNET(@red-0ne): Change `add-service` to `update-service` so the source owner can
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
$ pocketd tx service add-service "svc1" "service_one" 1 --keyring-backend test --from $(SERVICE_OWNER) --network=<network> --home $(POCKETD_HOME)`,
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
					return sharedtypes.ErrSharedInvalidComputeUnitsPerRelay.Wrapf("unable to parse as uint64: %s", args[2])
				}
			} else {
				fmt.Printf("Using default compute_units_per_relay: %d\n", types.DefaultComputeUnitsPerRelay)
			}

			serviceOwnerAddress := clientCtx.GetFromAddress().String()
			metadata, err := parseServiceMetadata(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddService(
				serviceOwnerAddress,
				serviceIdStr,
				serviceNameStr,
				computeUnitsPerRelay,
			)
			msg.Service.Metadata = metadata
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(FlagExperimentalMetadataBase64, "", "Base64 encoded experimental API specification for the service (mutually exclusive with --experimental--metadata-file)")
	cmd.Flags().String(FlagExperimentalMetadataFile, "", "Path to a file containing the experimental service API specification (mutually exclusive with --experimental--metadata-base64)")

	return cmd
}

const (
	FlagExperimentalMetadataBase64 = "experimental--metadata-base64"
	FlagExperimentalMetadataFile   = "experimental--metadata-file"
)

func parseServiceMetadata(cmd *cobra.Command) (*sharedtypes.Metadata, error) {
	metadataBase64, err := cmd.Flags().GetString(FlagExperimentalMetadataBase64)
	if err != nil {
		return nil, err
	}

	metadataFile, err := cmd.Flags().GetString(FlagExperimentalMetadataFile)
	if err != nil {
		return nil, err
	}

	if metadataBase64 != "" && metadataFile != "" {
		return nil, errors.New("--experimental--metadata-base64 and --experimental--metadata-file cannot be used together")
	}

	if metadataBase64 == "" && metadataFile == "" {
		return nil, nil
	}

	var apiSpecs []byte
	if metadataBase64 != "" {
		metadataBase64 = strings.TrimSpace(metadataBase64)
		apiSpecs, err = base64.StdEncoding.DecodeString(metadataBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode experimental--metadata-base64 value: %w", err)
		}
	} else {
		apiSpecs, err = os.ReadFile(metadataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read experimental metadata file %q: %w", metadataFile, err)
		}
	}

	if len(apiSpecs) > sharedtypes.MaxServiceMetadataSizeBytes {
		return nil, fmt.Errorf("experimental service metadata size %d exceeds max %d bytes", len(apiSpecs), sharedtypes.MaxServiceMetadataSizeBytes)
	}

	if len(apiSpecs) == 0 {
		return nil, errors.New("experimental service metadata cannot be empty")
	}

	return &sharedtypes.Metadata{ExperimentalApiSpecs: apiSpecs}, nil
}
