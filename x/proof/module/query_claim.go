package proof

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/proto/types/proof"
)

// AddPaginationFlagsToCmd adds common pagination flags to cmd
func AddClaimFilterFlags(cmd *cobra.Command) {
	cmd.Flags().Uint64(FlagSessionEndHeight, 0, "claims whose session ends at this height will be returned")
	cmd.Flags().String(FlagSessionId, "", "claims matching this session id will be returned")
	cmd.Flags().String(FlagSupplierAddress, "", "claims submitted by suppliers matching this address will be returned")
}

func CmdListClaims() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-claims",
		Short: "list all claims",
		Long: `List all the claims that the node being queried has in its state.

The claims can be optionally filtered by one of --session-end-height --session-id or --supplier-address flags

Example:
$ poktrolld q claim list-claims --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q claim list-claims --session-id <session_id> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q claim list-claims --session-end-height <session_end_height> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q claim list-claims --supplier-address <supplier_address> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			pageReq, pageErr := client.ReadPageRequest(cmd.Flags())
			if pageErr != nil {
				return pageErr
			}

			req := &proof.QueryAllClaimsRequest{
				Pagination: pageReq,
			}
			if err = updateClaimsFilter(cmd, req); err != nil {
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

			res, claimsErr := queryClient.AllClaims(cmd.Context(), req)
			if claimsErr != nil {
				return claimsErr
			}
			return clientCtx.PrintProto(res)
		},
	}

	AddClaimFilterFlags(cmd)
	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowClaim() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-claim <session_id> <supplier_addr>",
		Short: "shows a specific claim",
		Long: `List a specific claim that the node being queried has access to (if it still exists).

A unique claim can be defined via a ` + "`session_id`" + ` that the given ` + "`supplier`" + ` participated in.

` + "`Claims`" + ` are pruned, according to protocol parameters, some time after their respective ` + "`proof`" + ` has been submitted and any dispute window has elapsed.

This is done to minimize the rate at which state accumulates by eliminating claims as a long-term factor to persistence requirements.

Example:
$ poktrolld --home=$(POKTROLLD_HOME) q claim show-claims <session_id> <supplier_address> --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionId := args[0]
			supplierAddr := args[1]

			getClaimRequest := &proof.QueryGetClaimRequest{
				SessionId:       sessionId,
				SupplierAddress: supplierAddr,
			}
			if err := getClaimRequest.ValidateBasic(); err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := proof.NewQueryClient(clientCtx)

			res, err := queryClient.Claim(cmd.Context(), getClaimRequest)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// updateClaimsFilter updates the claims filter request based on the flags set provided
func updateClaimsFilter(cmd *cobra.Command, req *proof.QueryAllClaimsRequest) error {
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
		req.Filter = &proof.QueryAllClaimsRequest_SessionId{
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
		req.Filter = &proof.QueryAllClaimsRequest_SupplierAddress{
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
		req.Filter = &proof.QueryAllClaimsRequest_SessionEndHeight{
			SessionEndHeight: sessionEndHeight,
		}
		return nil
	}

	return nil
}
