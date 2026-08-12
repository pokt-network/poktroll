package gateway

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/cards"
)

// CmdValidateCard returns a command that checks a gateway card offline.
//
// DEV_NOTE: This lives under `tx` despite broadcasting nothing, alongside the command that
// publishes the card -- the same placement Cosmos uses for `tx validate-signatures`.
func CmdValidateCard() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-card <card_file>",
		Short: "Validate a gateway card against the Pocket Gateway Card schema",
		Long: `Validate a gateway card file offline, before spending gas to publish it.

The chain enforces the card's SIZE only -- it never parses the payload -- so this is the
only place a malformed card can be caught before it lands onchain. The schema is compiled
into the binary; nothing is fetched over the network.

Every problem found is reported, not just the first.

See docs/pocket_service_card.md for the schema.`,
		Example: `  pocketd tx gateway validate-card ./gateway-card.json`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Args are already validated by cobra, so anything failing below is a data
			// problem, not a usage problem -- do not dump help on top of the error.
			cmd.SilenceUsage = true

			card, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read card file %q: %w", args[0], err)
			}

			if err := cards.Validate(cards.KindGateway, card); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✅ valid gateway card: %s\n", cards.Summary(card))
			return nil
		},
	}

	return cmd
}
