package service

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/service/config"
	"github.com/pokt-network/poktroll/x/service/types"
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

Each service entry in the config must specify service_id and compute_units_per_relay.
The service_name field is optional and ignored (the chain does not support updating
service names for existing services).

The command queries the chain to verify:
  - Each service exists on-chain
  - The transaction signer is the service owner

Services whose on-chain compute_units_per_relay already matches are skipped.

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
  #     compute_units_per_relay: 25`,
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

				// Skip if on-chain compute_units_per_relay already matches the config.
				// NOTE: The chain's AddService keeper only updates ComputeUnitsPerRelay
				// and Metadata for existing services (not Name), so we only compare that field.
				// Ref: x/service/keeper/msg_server_add_service.go:55-65
				if onChainSvc.ComputeUnitsPerRelay == svcEntry.ComputeUnitsPerRelay {
					fmt.Fprintf(cmd.OutOrStdout(), "Skipping service %q: already up to date\n", svcEntry.ServiceId)
					continue
				}

				// Use on-chain name since the chain does not update names for existing services.
				// If the config provides a name, ignore it silently.
				serviceName := onChainSvc.Name

				msg := types.NewMsgAddService(
					signerAddress,
					svcEntry.ServiceId,
					serviceName,
					svcEntry.ComputeUnitsPerRelay,
				)
				if validateErr := msg.ValidateBasic(); validateErr != nil {
					return fmt.Errorf("invalid message for service %q: %w", svcEntry.ServiceId, validateErr)
				}

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

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
