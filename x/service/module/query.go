package service

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/cards"
	"github.com/pokt-network/poktroll/x/service/types"
)

// FlagRawCard prints the exact stored card bytes instead of re-indenting them.
const FlagRawCard = "raw"

// GetQueryCmd returns the cli query commands for this module.
//
// DEV_NOTE: The autocli-generated query commands (show-service, all-services,
// compute-units-per-relay-*, relay-mining-difficulty-*) are merged into this command by
// autocli, which only happens because the module's Query descriptor sets
// `EnhanceCustomCommand: true`. Removing that flag would silently delete every generated
// query command from the CLI, leaving only what is added below.
// Ref: cosmossdk.io/client/v2/autocli enhanceCustomCmd.
func (am AppModule) GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdShowServiceCard())

	return cmd
}

// CmdShowServiceCard prints a service's card, decoded.
func CmdShowServiceCard() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card <service_id>",
		Short: "Print a service's card, decoded",
		Long: `Print the card stored onchain for a service.

The card is stored as opaque bytes, so the regular query APIs return it base64 encoded
inside a JSON envelope. This decodes it for you:

  pocketd query service card eth

is equivalent to:

  pocketd query service show-service eth -o json | jq -r '.service.metadata.card' | base64 -d

Use --raw to emit the exact stored bytes with no re-indenting, which is what you want for
hashing, diffing, or feeding the card back into 'tx service add-service --card-file'.`,
		Example: `  pocketd query service card eth
  pocketd query service card eth --raw > card.json`,
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

			serviceId := args[0]
			queryClient := types.NewQueryClient(clientCtx)

			// Dehydrated defaults to false, so the card is included.
			res, err := queryClient.Service(cmd.Context(), &types.QueryGetServiceRequest{Id: serviceId})
			if err != nil {
				return err
			}

			// GetService returns a value, not a pointer, so it needs a local before the
			// pointer-receiver accessors can be used.
			service := res.GetService()
			card := service.GetMetadata().GetCard()
			if len(card) == 0 {
				return fmt.Errorf("service %q has no card", serviceId)
			}

			return cards.Fprint(cmd.OutOrStdout(), card, raw)
		},
	}

	cmd.Flags().Bool(FlagRawCard, false, "Print the exact stored bytes instead of re-indenting the JSON")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
