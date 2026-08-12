package gateway

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/cards"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// FlagRawCard prints the exact stored card bytes instead of re-indenting them.
const FlagRawCard = "raw"

// CmdShowGatewayCard prints a gateway's card, decoded.
func CmdShowGatewayCard() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card <gateway_address>",
		Short: "Print a gateway's card, decoded",
		Long: `Print the card stored onchain for a gateway.

The card is stored as opaque bytes, so the regular query APIs return it base64 encoded
inside a JSON envelope. This decodes it for you.

Use --raw to emit the exact stored bytes with no re-indenting, which is what you want for
hashing, diffing, or feeding the card back into
'tx gateway update-gateway-metadata --card-file'.`,
		Example: `  pocketd query gateway card pokt1...
  pocketd query gateway card pokt1... --raw > gateway-card.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Args are already validated by cobra; anything below is a data or network
			// problem, not a usage problem.
			cmd.SilenceUsage = true

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			raw, err := cmd.Flags().GetBool(FlagRawCard)
			if err != nil {
				return err
			}

			address := args[0]
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Gateway(cmd.Context(), &types.QueryGetGatewayRequest{Address: address})
			if err != nil {
				return err
			}

			// GetGateway returns a value, not a pointer, so it needs a local before the
			// pointer-receiver accessors can be used.
			gateway := res.GetGateway()
			card := gateway.GetMetadata().GetCard()
			if len(card) == 0 {
				return fmt.Errorf("gateway %q has no card", address)
			}

			return cards.Fprint(cmd.OutOrStdout(), card, raw)
		},
	}

	cmd.Flags().Bool(FlagRawCard, false, "Print the exact stored bytes instead of re-indenting the JSON")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
