package proof

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// AddPaginationFlagsToCmd adds common pagination flags to cmd
func AddClaimFilterFlags(cmd *cobra.Command) {
	cmd.Flags().Uint64(FlagSessionEndHeight, 0, "claims whose session ends at this height will be returned")
	cmd.Flags().String(FlagSessionId, "", "claims matching this session id will be returned")
	cmd.Flags().String(FlagSupplierOperatorAddress, "", "claims submitted by suppliers matching this operator address will be returned")
}

func CmdListClaims() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-claims",
		Short: "list all claims",
		Long: `List all the claims that the node being queried has in its state.

The claims can be optionally filtered by one of --session-end-height --session-id or --supplier-operator-address flags

Example:
$ poktrolld q claim list-claims --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q claim list-claims --session-id <session_id> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q claim list-claims --session-end-height <session_end_height> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)
$ poktrolld q claim list-claims --supplier-operator-address <supplier_operator_address> --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			pageReq, pageErr := client.ReadPageRequest(cmd.Flags())
			if pageErr != nil {
				return pageErr
			}

			req := &types.QueryAllClaimsRequest{
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
			queryClient := types.NewQueryClient(clientCtx)

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
		Use:   "show-claim <session_id> <supplier_operator_addr>",
		Short: "shows a specific claim",
		Long: `List a specific claim that the node being queried has access to (if it still exists).

A unique claim can be defined via a ` + "`session_id`" + ` that the given ` + "`supplier`" + ` participated in.

` + "`Claims`" + ` are pruned, according to protocol parameters, some time after their respective ` + "`proof`" + ` has been submitted and any dispute window has elapsed.

This is done to minimize the rate at which state accumulates by eliminating claims as a long-term factor to persistence requirements.

Example:
$ poktrolld --home=$(POKTROLLD_HOME) q claim show-claims <session_id> <supplier_operator_address> --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionId := args[0]
			supplierOperatorAddr := args[1]

			getClaimRequest := &types.QueryGetClaimRequest{
				SessionId:               sessionId,
				SupplierOperatorAddress: supplierOperatorAddr,
			}
			if err := getClaimRequest.ValidateBasic(); err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

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
func updateClaimsFilter(cmd *cobra.Command, req *types.QueryAllClaimsRequest) error {
	sessionId, _ := cmd.Flags().GetString(FlagSessionId)
	supplierOperatorAddr, _ := cmd.Flags().GetString(FlagSupplierOperatorAddress)
	sessionEndHeight, _ := cmd.Flags().GetUint64(FlagSessionEndHeight)

	// Preparing a shared error in case more than one flag was set
	err := fmt.Errorf(
		"can only specify one flag filter but got sessionId (%s), supplierOperatorAddr (%s) and sessionEngHeight (%d)",
		sessionId,
		supplierOperatorAddr,
		sessionEndHeight,
	)

	// Use the session id as the filter
	if sessionId != "" {
		// If the session id is set, then the other flags must not be set
		if supplierOperatorAddr != "" || sessionEndHeight > 0 {
			return err
		}
		// Set the session id filter
		req.Filter = &types.QueryAllClaimsRequest_SessionId{
			SessionId: sessionId,
		}
		return nil
	}

	// Use the supplier operator address as the filter
	if supplierOperatorAddr != "" {
		// If the supplier operator address is set, then the other flags must not be set
		if sessionId != "" || sessionEndHeight > 0 {
			return err
		}
		// Set the supplier operator address filter
		req.Filter = &types.QueryAllClaimsRequest_SupplierOperatorAddress{
			SupplierOperatorAddress: supplierOperatorAddr,
		}
		return nil
	}

	// Use the session end height as the filter
	if sessionEndHeight > 0 {
		// If the session end height is set, then the other flags must not be set
		if sessionId != "" || supplierOperatorAddr != "" {
			return err
		}
		// Set the session end height filter
		req.Filter = &types.QueryAllClaimsRequest_SessionEndHeight{
			SessionEndHeight: sessionEndHeight,
		}
		return nil
	}

	return nil
}
