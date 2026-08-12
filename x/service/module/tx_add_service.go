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

	"github.com/pokt-network/poktroll/pkg/cards"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ = strconv.Itoa(0)

// CmdAddService returns a CLI command for adding or updating a service on the network.
// TODO_POST_MAINNET(@red-0ne): Change `add-service` to `update-service` so the source owner can
// update the compute units per relay for an existing service. Make it possible
// to update a service (e.g. update # of compute units per relay). This will require
// search for all variations of `AddService` in the codebase (filenames, helpers, etc...),
// ensuring that only the owner can update it on chain, and tackling some of the tests in `service.feature`.
func CmdAddService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("add-service <service_id> <service_name> [compute_units_per_relay: default=%d]", types.DefaultComputeUnitsPerRelay),
		Short: "Add or update a service on the network",
		Long: `Add a new service or update an existing service on the network.

This command allows any actor to add a new service or update an existing one (if they are the owner).
Services are uniquely identified by their ID and can optionally carry a service card: a small,
self-describing JSON document (limited to 256 KiB). See docs/pocket_service_card.md.

The service ID MUST be unique but the service name doesn't have to be.
Only the service owner can update an existing service.`,
		Example: `  # Add a basic service without a card
  pocketd tx service add-service "svc1" "My Service" 10 --from owner

  # Add a service with a card from a file
  pocketd tx service add-service "svc1" "My Service" 10 \
    --card-file ./card.json --from owner

  # Add a service with a base64-encoded card
  pocketd tx service add-service "svc1" "My Service" 10 \
    --card-base64 $(base64 -w0 ./card.json) --from owner

  # Update an existing service's compute units and card
  pocketd tx service add-service "svc1" "My Service" 20 \
    --card-file ./card-v2.json --from owner`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Args are already validated by cobra, so anything failing below (a malformed
			// card, a missing key, a broadcast failure) is not a usage problem.
			cmd.SilenceUsage = true

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
	cmd.Flags().String(
		FlagCardBase64,
		"",
		"Base64-encoded service card (JSON) for the service. "+
			"Limited to 256 KiB when decoded. Mutually exclusive with --card-file.",
	)
	cmd.Flags().String(
		FlagCardFile,
		"",
		"Path to a file containing the service card (JSON). "+
			"Limited to 256 KiB. Mutually exclusive with --card-base64.",
	)

	cmd.Flags().Bool(
		FlagSkipCardValidation,
		false,
		"Publish the card without validating it against the Pocket Service Card schema. "+
			"The chain does not parse the payload, so this lets you store something that is not a card.",
	)

	// Deprecated aliases, kept so existing tooling and scripts keep working.
	cmd.Flags().String(FlagExperimentalMetadataBase64, "", "Deprecated: use --card-base64.")
	cmd.Flags().String(FlagExperimentalMetadataFile, "", "Deprecated: use --card-file.")
	_ = cmd.Flags().MarkDeprecated(FlagExperimentalMetadataBase64, "use --card-base64")
	_ = cmd.Flags().MarkDeprecated(FlagExperimentalMetadataFile, "use --card-file")

	return cmd
}

const (
	// FlagSkipCardValidation opts out of client-side card schema validation.
	FlagSkipCardValidation = "skip-card-validation"

	// FlagCardBase64 is the flag name for providing a base64-encoded service card.
	FlagCardBase64 = "card-base64"

	// FlagCardFile is the flag name for providing a file path containing a service card.
	FlagCardFile = "card-file"

	// FlagExperimentalMetadataBase64 is the deprecated alias for FlagCardBase64.
	FlagExperimentalMetadataBase64 = "experimental-metadata-base64"

	// FlagExperimentalMetadataFile is the deprecated alias for FlagCardFile.
	FlagExperimentalMetadataFile = "experimental-metadata-file"
)

// parseServiceMetadata parses the service card from command-line flags.
// It supports two mutually exclusive ways of providing the card:
// 1. --card-base64: Base64-encoded card
// 2. --card-file: Path to a file containing the card
//
// The deprecated --experimental-metadata-{base64,file} aliases are still accepted.
//
// The card must not exceed 256 KiB when decoded. See docs/pocket_service_card.md.
//
// Returns:
//   - *sharedtypes.Metadata: The parsed card, or nil if none was provided
//   - error: An error if parsing fails, flags conflict, or size limits are exceeded
func parseServiceMetadata(cmd *cobra.Command) (*sharedtypes.Metadata, error) {
	// Resolve each source from its current flag, falling back to the deprecated alias.
	cardBase64, err := flagWithDeprecatedAlias(cmd, FlagCardBase64, FlagExperimentalMetadataBase64)
	if err != nil {
		return nil, err
	}

	cardFile, err := flagWithDeprecatedAlias(cmd, FlagCardFile, FlagExperimentalMetadataFile)
	if err != nil {
		return nil, err
	}

	// Ensure only one card source is provided
	if cardBase64 != "" && cardFile != "" {
		return nil, errors.New("--card-base64 and --card-file cannot be used together")
	}

	// If no card is provided, return nil (the card is optional)
	if cardBase64 == "" && cardFile == "" {
		return nil, nil
	}

	// Parse the card from either the base64 string or the file
	var card []byte
	if cardBase64 != "" {
		cardBase64 = strings.TrimSpace(cardBase64)
		card, err = base64.StdEncoding.DecodeString(cardBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode card-base64 value: %w", err)
		}
	} else {
		card, err = os.ReadFile(cardFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read card file %q: %w", cardFile, err)
		}
	}

	// Validate the card size does not exceed the maximum allowed (256 KiB)
	if len(card) > sharedtypes.MaxServiceMetadataSizeBytes {
		// TODO_FUTURE: Add validation hints suggesting the user point at a spec URL instead.
		return nil, fmt.Errorf("service card size %d exceeds max %d bytes (256 KiB)", len(card), sharedtypes.MaxServiceMetadataSizeBytes)
	}

	// Ensure the card is not empty (if provided, it must contain data)
	if len(card) == 0 {
		return nil, errors.New("service card cannot be empty")
	}

	// Validate against the card schema unless explicitly skipped.
	//
	// The chain enforces size only and never parses this payload, so this is the ONLY
	// place a malformed card is caught before it costs gas and lands onchain.
	skipValidation, err := cmd.Flags().GetBool(FlagSkipCardValidation)
	if err != nil {
		return nil, err
	}
	if !skipValidation {
		if err := cards.Validate(cards.KindService, card); err != nil {
			return nil, fmt.Errorf("%w\n\nRe-run with --%s to publish it anyway", err, FlagSkipCardValidation)
		}
	}

	return &sharedtypes.Metadata{Card: card}, nil
}

// flagWithDeprecatedAlias returns the value of name, falling back to deprecatedName when
// name is unset. Setting both is an error rather than a silent precedence rule.
func flagWithDeprecatedAlias(cmd *cobra.Command, name, deprecatedName string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}

	deprecatedValue, err := cmd.Flags().GetString(deprecatedName)
	if err != nil {
		return "", err
	}

	if value != "" && deprecatedValue != "" {
		return "", fmt.Errorf("--%s and its deprecated alias --%s cannot be used together", name, deprecatedName)
	}

	if value == "" {
		return deprecatedValue, nil
	}

	return value, nil
}
