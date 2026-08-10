package gateway

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// FlagDehydrated omits gateway cards from a query response.
const FlagDehydrated = "dehydrated"

func CmdListGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-gateway",
		Short: "list all gateways",
		Long: `List all the gateways that the node being queried has in its state.

Cards are included by default. Pass --dehydrated to omit them: a card can be up to 256 KiB,
so enumerating gateways that carry one can produce a response large enough to exceed the
default 4 MB gRPC message limit. Read an individual card with 'query gateway card <address>'.

Example:
$ pocketd q gateway list-gateway --network=<network> --home $(POCKETD_HOME)
$ pocketd q gateway list-gateway --dehydrated --network=<network>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			dehydrated, err := cmd.Flags().GetBool(FlagDehydrated)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllGatewaysRequest{
				Pagination: pageReq,
				Dehydrated: dehydrated,
			}

			res, err := queryClient.AllGateways(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	cmd.Flags().Bool(FlagDehydrated, false, "Omit gateway cards from the response")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowGateway() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-gateway <gateway_address>",
		Short: "shows a gateway",
		Long: `Show a single gateway.

The gateway's card is included by default. Pass --dehydrated to omit it.

Example:
$ pocketd q gateway show-gateway pokt1... --network=<network> --home $(POCKETD_HOME)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argAddress := args[0]

			dehydrated, err := cmd.Flags().GetBool(FlagDehydrated)
			if err != nil {
				return err
			}

			params := &types.QueryGetGatewayRequest{
				Address:    argAddress,
				Dehydrated: dehydrated,
			}

			res, err := queryClient.Gateway(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(FlagDehydrated, false, "Omit the gateway card from the response")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
