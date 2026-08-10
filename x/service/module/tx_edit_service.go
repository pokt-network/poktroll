package service

import (
	"bytes"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/cards"
	"github.com/pokt-network/poktroll/x/service/config"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const FlagDisableBatchMsgs = "disable-batch-msgs"

// CmdEditService returns a CLI command for batch-editing existing services from
// a YAML config file. It queries the chain to verify each service exists and is
// owned by the signer, skips services whose values already match on-chain, and
// submits the remaining updates as a single batched transaction (or individual
// transactions when --disable-batch-msgs is set).
func CmdEditService() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-service --config <config-file>",
		Short: "Batch-update existing services from a YAML config file",
		Long: `Update one or more existing services on the network from a YAML config file.

Each service entry must specify service_id and compute_units_per_relay, and may optionally
name a card_file: a Pocket Service Card to publish for that service
(see docs/pocket_service_card.md). Cards are validated against the schema before anything
is broadcast; --skip-card-validation opts out.

The service_name field is optional and ignored (the chain does not support updating
service names for existing services).

The command queries the chain to verify:
  - Each service exists on-chain
  - The transaction signer is the service owner

A service is skipped only when BOTH its compute_units_per_relay and its card already match
what is on-chain. An entry that names no card_file leaves the stored card untouched.

Card comparison is byte-exact against what is stored, so reformatting a card file counts
as a change even when the JSON is semantically identical.

By default, all updates are submitted as a single batched transaction. Use
--disable-batch-msgs to send individual transactions instead.`,
		Example: `  # Update services from a config file
  pocketd tx service edit-service --config services.yaml --from owner --fees 300upokt

  # Send individual transactions instead of a batch
  pocketd tx service edit-service --config services.yaml --disable-batch-msgs --from owner

  # Example services.yaml:
  # services:
  #   - service_id: svc1
  #     compute_units_per_relay: 15
  #   - service_id: svc2
  #     compute_units_per_relay: 25
  #     card_file: ./cards/svc2.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFile, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			configContent, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file %q: %w", configFile, err)
			}

			editConfig, err := config.ParseEditServiceConfig(configContent)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			signerAddress := clientCtx.GetFromAddress().String()
			queryClient := types.NewQueryClient(clientCtx)

			var msgs []sdk.Msg
			for _, svcEntry := range editConfig.Services {
				// Query the on-chain service to verify existence and ownership.
				resp, queryErr := queryClient.Service(
					cmd.Context(),
					&types.QueryGetServiceRequest{Id: svcEntry.ServiceId},
				)
				if queryErr != nil {
					return fmt.Errorf("service %q not found on-chain: %w", svcEntry.ServiceId, queryErr)
				}

				onChainSvc := resp.GetService()

				// Verify the signer owns the service.
				if onChainSvc.OwnerAddress != signerAddress {
					return fmt.Errorf(
						"signer %q is not the owner of service %q (owner: %q)",
						signerAddress, svcEntry.ServiceId, onChainSvc.OwnerAddress,
					)
				}

				// Load the card for this entry, if the config names one.
				desiredCard, cardErr := loadEntryCard(cmd, svcEntry)
				if cardErr != nil {
					return cardErr
				}

				// Decide what actually needs to change.
				//
				// DEV_NOTE: This MUST consider the card as well as cupr. Comparing cupr alone
				// -- as this command originally did -- silently drops card-only edits: the
				// entry looks "already up to date" and is skipped, so the new card is never
				// published and the command reports success.
				cuprChanged := onChainSvc.ComputeUnitsPerRelay != svcEntry.ComputeUnitsPerRelay
				cardChanged := desiredCard != nil && !bytes.Equal(onChainSvc.GetMetadata().GetCard(), desiredCard)

				if !cuprChanged && !cardChanged {
					fmt.Fprintf(cmd.OutOrStdout(), "Skipping service %q: already up to date\n", svcEntry.ServiceId)
					continue
				}

				// Re-send the on-chain name so this command never renames a service as a side
				// effect. The keeper DOES overwrite Name on update, so omitting it here would
				// blank it out. If the config provides a name, ignore it silently.
				serviceName := onChainSvc.Name

				msg := types.NewMsgAddService(
					signerAddress,
					svcEntry.ServiceId,
					serviceName,
					svcEntry.ComputeUnitsPerRelay,
				)

				// Attach the card ONLY when the config named one. Leaving Metadata nil means
				// the keeper preserves whatever is already stored, so a cupr-only edit never
				// disturbs an existing card.
				if desiredCard != nil {
					msg.Service.Metadata = &sharedtypes.Metadata{Card: desiredCard}
				}

				if validateErr := msg.ValidateBasic(); validateErr != nil {
					return fmt.Errorf("invalid message for service %q: %w", svcEntry.ServiceId, validateErr)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Updating service %q: %s\n",
					svcEntry.ServiceId, describeChanges(cuprChanged, cardChanged))

				msgs = append(msgs, msg)
			}

			if len(msgs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "All services are already up to date, nothing to do.")
				return nil
			}

			disableBatch, err := cmd.Flags().GetBool(FlagDisableBatchMsgs)
			if err != nil {
				return err
			}

			if disableBatch {
				// Create the factory once and manually increment the sequence
				// for each tx to avoid sequence mismatch errors when multiple
				// txs land in the same block.
				txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
				if err != nil {
					return err
				}
				txf, err = txf.Prepare(clientCtx)
				if err != nil {
					return err
				}

				for i, msg := range msgs {
					currentTxf := txf.WithSequence(txf.Sequence() + uint64(i))
					if err := tx.GenerateOrBroadcastTxWithFactory(clientCtx, currentTxf, msg); err != nil {
						return err
					}
				}
				return nil
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	cmd.Flags().String("config", "", "Path to the YAML config file with service definitions (required)")
	_ = cmd.MarkFlagRequired("config")
	cmd.Flags().Bool(FlagDisableBatchMsgs, false, "Send individual transactions instead of a single batch")
	cmd.Flags().Bool(
		FlagSkipCardValidation,
		false,
		"Publish cards without validating them against the Pocket Service Card schema",
	)

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// loadEntryCard reads and validates the card named by a config entry, if any.
// It returns nil when the entry names no card, which the caller treats as
// "leave the stored card alone".
func loadEntryCard(cmd *cobra.Command, svcEntry *config.YAMLServiceEntry) ([]byte, error) {
	if svcEntry.CardFile == "" {
		return nil, nil
	}

	card, err := os.ReadFile(svcEntry.CardFile)
	if err != nil {
		return nil, fmt.Errorf("service %q: failed to read card_file %q: %w",
			svcEntry.ServiceId, svcEntry.CardFile, err)
	}

	skipValidation, err := cmd.Flags().GetBool(FlagSkipCardValidation)
	if err != nil {
		return nil, err
	}

	if !skipValidation {
		if validateErr := cards.Validate(cards.KindService, card); validateErr != nil {
			return nil, fmt.Errorf("service %q: %w\n\nRe-run with --%s to publish it anyway",
				svcEntry.ServiceId, validateErr, FlagSkipCardValidation)
		}
	}

	return card, nil
}

// describeChanges renders which fields an update is actually changing, so a batch run says
// what it did rather than just how many messages it sent.
func describeChanges(cuprChanged, cardChanged bool) string {
	switch {
	case cuprChanged && cardChanged:
		return "compute_units_per_relay and card"
	case cuprChanged:
		return "compute_units_per_relay"
	default:
		return "card"
	}
}
