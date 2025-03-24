package gateway

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/pocket/x/gateway/types"
)

func CmdListGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-gateway",
		Short: "list all gateways",
		Long: `List all the gateways that the node being queried has in its state.

Example:
$ pocketd q gateway list-gateway --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllGatewaysRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.AllGateways(cmd.Context(), params)
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

func CmdShowGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-gateway <gateway_address>",
		Short: "shows a gateway",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argAddress := args[0]

			params := &types.QueryGetGatewayRequest{
				Address: argAddress,
			}

			res, err := queryClient.Gateway(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
