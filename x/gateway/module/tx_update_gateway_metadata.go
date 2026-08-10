package gateway

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/cards"
	"github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	// FlagCardBase64 is the flag name for providing a base64-encoded gateway card.
	FlagCardBase64 = "card-base64"

	// FlagCardFile is the flag name for providing a file path containing a gateway card.
	FlagCardFile = "card-file"

	// FlagSkipCardValidation opts out of client-side card schema validation.
	FlagSkipCardValidation = "skip-card-validation"
)

func CmdUpdateGatewayMetadata() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "update-gateway-metadata",
		Short: "Set a staked gateway's card",
		Long: `Set the card of the gateway specified by the 'from' address.

A gateway card is a small, self-describing JSON document (limited to 256 KiB) using the
same container and conventions as a service card. See docs/pocket_service_card.md.

This never touches the gateway's stake: unlike stake-gateway, which requires escrowing
additional POKT on every call, updating a card costs only gas. The gateway must already
be staked.

Example:
$ pocketd tx gateway update-gateway-metadata --card-file ./gateway-card.json \
    --keyring-backend test --from $(GATEWAY) --network=<network> --home $(POCKETD_HOME)`,
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			// Anything failing below is a data or environment problem, not a usage problem.
			cmd.SilenceUsage = true

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			metadata, err := parseGatewayCard(cmd)
			if err != nil {
				return err
			}
			if metadata == nil {
				return errors.New("one of --card-base64 or --card-file is required")
			}

			msg := types.NewMsgUpdateGatewayMetadata(
				clientCtx.GetFromAddress().String(),
				metadata,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(
		FlagCardBase64,
		"",
		"Base64-encoded gateway card (JSON). Limited to 256 KiB when decoded. "+
			"Mutually exclusive with --card-file.",
	)
	cmd.Flags().String(
		FlagCardFile,
		"",
		"Path to a file containing the gateway card (JSON). Limited to 256 KiB. "+
			"Mutually exclusive with --card-base64.",
	)
	cmd.Flags().Bool(
		FlagSkipCardValidation,
		false,
		"Publish the card without validating it against the Pocket Gateway Card schema. "+
			"The chain does not parse the payload, so this lets you store something that is not a card.",
	)

	return cmd
}

// parseGatewayCard reads the gateway card from the mutually exclusive --card-base64 and
// --card-file flags. It returns nil when neither is set.
func parseGatewayCard(cmd *cobra.Command) (*sharedtypes.Metadata, error) {
	cardBase64, err := cmd.Flags().GetString(FlagCardBase64)
	if err != nil {
		return nil, err
	}

	cardFile, err := cmd.Flags().GetString(FlagCardFile)
	if err != nil {
		return nil, err
	}

	if cardBase64 != "" && cardFile != "" {
		return nil, errors.New("--card-base64 and --card-file cannot be used together")
	}

	if cardBase64 == "" && cardFile == "" {
		return nil, nil
	}

	var card []byte
	if cardBase64 != "" {
		card, err = base64.StdEncoding.DecodeString(strings.TrimSpace(cardBase64))
		if err != nil {
			return nil, fmt.Errorf("failed to decode card-base64 value: %w", err)
		}
	} else {
		card, err = os.ReadFile(cardFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read card file %q: %w", cardFile, err)
		}
	}

	// Size is checked again onchain; failing here avoids broadcasting a doomed tx.
	if len(card) > sharedtypes.MaxServiceMetadataSizeBytes {
		return nil, fmt.Errorf(
			"gateway card size %d exceeds max %d bytes (256 KiB)",
			len(card), sharedtypes.MaxServiceMetadataSizeBytes,
		)
	}

	if len(card) == 0 {
		return nil, errors.New("gateway card cannot be empty")
	}

	// Validate against the card schema unless explicitly skipped. The chain enforces size
	// only and never parses this payload.
	skipValidation, err := cmd.Flags().GetBool(FlagSkipCardValidation)
	if err != nil {
		return nil, err
	}
	if !skipValidation {
		if err := cards.Validate(cards.KindGateway, card); err != nil {
			return nil, fmt.Errorf("%w\n\nRe-run with --%s to publish it anyway", err, FlagSkipCardValidation)
		}
	}

	return &sharedtypes.Metadata{Card: card}, nil
}
