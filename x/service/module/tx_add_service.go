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

// CmdAddService returns a CLI command for adding or updating a service on the network.
//
// This command allows any actor to add a new service or update an existing one (if they are the owner).
// Services are uniquely identified by their ID and can optionally include experimental metadata
// such as OpenAPI or OpenRPC specifications.
//
// Usage:
//
//	pocketd tx service add-service <service_id> <service_name> [compute_units_per_relay]
//	  [--experimental--metadata-file <path> | --experimental--metadata-base64 <base64>]
//	  --from <owner> [flags]
//
// Examples:
//
//	# Add a service without metadata
//	pocketd tx service add-service "svc1" "My Service" 10 --from owner
//
//	# Add a service with metadata from file
//	pocketd tx service add-service "svc1" "My Service" 10 \
//	  --experimental--metadata-file ./openapi.json --from owner
//
//	# Update an existing service's compute units and metadata
//	pocketd tx service add-service "svc1" "My Service" 20 \
//	  --experimental--metadata-file ./openapi-v2.json --from owner
//
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
			// Parse required arguments
			serviceIdStr := args[0]
			serviceNameStr := args[1]

			// Get the client context for transaction signing
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Parse optional compute units per relay argument, or use default
			computeUnitsPerRelay := types.DefaultComputeUnitsPerRelay
			if len(args) > 2 {
				computeUnitsPerRelay, err = strconv.ParseUint(args[2], 10, 64)
				if err != nil {
					return sharedtypes.ErrSharedInvalidComputeUnitsPerRelay.Wrapf("unable to parse as uint64: %s", args[2])
				}
			} else {
				fmt.Printf("Using default compute_units_per_relay: %d\n", types.DefaultComputeUnitsPerRelay)
			}

			// Get the service owner address from the transaction signer
			serviceOwnerAddress := clientCtx.GetFromAddress().String()

			// Parse optional experimental metadata from flags
			metadata, err := parseServiceMetadata(cmd)
			if err != nil {
				return err
			}

			// Create the MsgAddService with the parsed parameters
			msg := types.NewMsgAddService(
				serviceOwnerAddress,
				serviceIdStr,
				serviceNameStr,
				computeUnitsPerRelay,
			)

			// Attach metadata to the service if provided
			msg.Service.Metadata = metadata

			// Validate the message before broadcasting
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			// Generate and broadcast the transaction
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(FlagExperimentalMetadataBase64, "", "Base64 encoded experimental API specification for the service (mutually exclusive with --experimental--metadata-file)")
	cmd.Flags().String(FlagExperimentalMetadataFile, "", "Path to a file containing the experimental service API specification (mutually exclusive with --experimental--metadata-base64)")

	return cmd
}

const (
	// FlagExperimentalMetadataBase64 is the flag name for providing base64-encoded
	// experimental service metadata (API specifications like OpenAPI, OpenRPC, etc.)
	FlagExperimentalMetadataBase64 = "experimental--metadata-base64"

	// FlagExperimentalMetadataFile is the flag name for providing a file path
	// containing experimental service metadata (API specifications)
	FlagExperimentalMetadataFile = "experimental--metadata-file"
)

// parseServiceMetadata parses experimental service metadata from command-line flags.
// It supports two mutually exclusive ways of providing metadata:
// 1. --experimental--metadata-base64: Base64-encoded metadata string
// 2. --experimental--metadata-file: Path to a file containing the metadata
//
// The metadata payload must not exceed 100 KiB when decoded. This is typically used
// to attach API specifications (OpenAPI, OpenRPC, etc.) to a service.
//
// Returns:
//   - *sharedtypes.Metadata: The parsed metadata, or nil if no metadata was provided
//   - error: An error if parsing fails, flags conflict, or size limits are exceeded
func parseServiceMetadata(cmd *cobra.Command) (*sharedtypes.Metadata, error) {
	// Retrieve the base64-encoded metadata flag value
	metadataBase64, err := cmd.Flags().GetString(FlagExperimentalMetadataBase64)
	if err != nil {
		return nil, err
	}

	// Retrieve the metadata file path flag value
	metadataFile, err := cmd.Flags().GetString(FlagExperimentalMetadataFile)
	if err != nil {
		return nil, err
	}

	// Ensure only one metadata source is provided
	if metadataBase64 != "" && metadataFile != "" {
		return nil, errors.New("--experimental--metadata-base64 and --experimental--metadata-file cannot be used together")
	}

	// If no metadata is provided, return nil (metadata is optional)
	if metadataBase64 == "" && metadataFile == "" {
		return nil, nil
	}

	// Parse the metadata from either base64 string or file
	var apiSpecs []byte
	if metadataBase64 != "" {
		// Decode base64-encoded metadata
		metadataBase64 = strings.TrimSpace(metadataBase64)
		apiSpecs, err = base64.StdEncoding.DecodeString(metadataBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode experimental--metadata-base64 value: %w", err)
		}
	} else {
		// Read metadata from file
		apiSpecs, err = os.ReadFile(metadataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read experimental metadata file %q: %w", metadataFile, err)
		}
	}

	// Validate metadata size does not exceed the maximum allowed (100 KiB)
	if len(apiSpecs) > sharedtypes.MaxServiceMetadataSizeBytes {
		return nil, fmt.Errorf("experimental service metadata size %d exceeds max %d bytes", len(apiSpecs), sharedtypes.MaxServiceMetadataSizeBytes)
	}

	// Ensure metadata is not empty (if provided, it must contain data)
	if len(apiSpecs) == 0 {
		return nil, errors.New("experimental service metadata cannot be empty")
	}

	return &sharedtypes.Metadata{ExperimentalApiSpecs: apiSpecs}, nil
}
