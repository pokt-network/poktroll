package supplier

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/proto/types/supplier"
)

func CmdListSuppliers() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-supplier",
		Short: "list all supplier",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := supplier.NewQueryClient(clientCtx)

			params := &supplier.QueryAllSuppliersRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.AllSuppliers(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowSupplier() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-supplier <supplier_address>",
		Short: "shows a supplier",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := supplier.NewQueryClient(clientCtx)

			argAddress := args[0]

			params := &supplier.QueryGetSupplierRequest{
				Address: argAddress,
			}

			res, err := queryClient.Supplier(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
