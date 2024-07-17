package proof

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/proto/types/proof"
)

// AddProofFilterFlagsToCmd adds common pagination flags to cmd
func AddProofFilterFlagsToCmd(cmd *cobra.Command) {
	cmd.Flags().Uint64(FlagSessionEndHeight, 0, "proofs whose session ends at this height will be returned")
	cmd.Flags().String(FlagSessionId, "", "proofs matching this session id will be returned")
	cmd.Flags().String(FlagSupplierAddress, "", "proofs submitted by suppliers matching this address will be returned")
}

func CmdListProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-proofs",
		Short: "list all proofs",
		Long: `List all the proofs that the node being queried has in its state.

The proofs can be optionally filtered by one of --session-end-height --session-id or --supplier-address flags

Example:
$ poktrolld q proof list-proofs --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q proof list-proofs --session-id <session_id> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q proof list-proofs --session-end-height <session_end_height> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q proof list-proofs --supplier-address <supplier_address> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &proof.QueryAllProofsRequest{
				Pagination: pageReq,
			}
			if err = updateProofsFilter(cmd, req); err != nil {
				return err
			}
			if err = req.ValidateBasic(); err != nil {
				return err
			}

			clientCtx, ctxErr := client.GetClientQueryContext(cmd)
			if ctxErr != nil {
				return ctxErr
			}
			queryClient := proof.NewQueryClient(clientCtx)

			res, proofsErr := queryClient.AllProofs(cmd.Context(), req)
			if proofsErr != nil {
				return proofsErr
			}

			return clientCtx.PrintProto(res)
		},
	}

	AddProofFilterFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-proof <session_id> <supplier_addr>",
		Short: "shows a specific proof",
		Long: `List a specific proof that the node being queried has access to.

A unique proof can be defined via a session_id that a given supplier participated in.

Example:
$ poktrolld --home=$(POKTROLLD_HOME) q proof show-proofs <session_id> <supplier_address> --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			sessionId := args[0]
			supplierAddr := args[1]

			getProofRequest := &proof.QueryGetProofRequest{
				SessionId:       sessionId,
				SupplierAddress: supplierAddr,
			}
			if err = getProofRequest.ValidateBasic(); err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := proof.NewQueryClient(clientCtx)

			res, err := queryClient.Proof(cmd.Context(), getProofRequest)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// updateProofsFilter updates the proofs filter request based on the flags set provided
func updateProofsFilter(cmd *cobra.Command, req *proof.QueryAllProofsRequest) error {
	sessionId, _ := cmd.Flags().GetString(FlagSessionId)
	supplierAddr, _ := cmd.Flags().GetString(FlagSupplierAddress)
	sessionEndHeight, _ := cmd.Flags().GetUint64(FlagSessionEndHeight)

	// Preparing a shared error in case more than one flag was set
	err := fmt.Errorf("can only specify one flag filter but got sessionId (%s), supplierAddr (%s) and sessionEngHeight (%d)", sessionId, supplierAddr, sessionEndHeight)

	// Use the session id as the filter
	if sessionId != "" {
		// If the session id is set, then the other flags must not be set
		if supplierAddr != "" || sessionEndHeight > 0 {
			return err
		}
		// Set the session id filter
		req.Filter = &proof.QueryAllProofsRequest_SessionId{
			SessionId: sessionId,
		}
		return nil
	}

	// Use the supplier address as the filter
	if supplierAddr != "" {
		// If the supplier address is set, then the other flags must not be set
		if sessionId != "" || sessionEndHeight > 0 {
			return err
		}
		// Set the supplier address filter
		req.Filter = &proof.QueryAllProofsRequest_SupplierAddress{
			SupplierAddress: supplierAddr,
		}
		return nil
	}

	// Use the session end height as the filter
	if sessionEndHeight > 0 {
		// If the session end height is set, then the other flags must not be set
		if sessionId != "" || supplierAddr != "" {
			return err
		}
		// Set the session end height filter
		req.Filter = &proof.QueryAllProofsRequest_SessionEndHeight{
			SessionEndHeight: sessionEndHeight,
		}
		return nil
	}

	return nil
}
