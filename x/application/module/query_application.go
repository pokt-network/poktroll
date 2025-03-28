package application

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/application/types"
)

func CmdListApplication() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-application",
		Short: "list all application",
		Long: `List all the applications that staked in the network.

Example:
$ pocketd q application list-application --node $(POCKET_NODE) --home $(POCKETD_HOME)`,
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

			params := &types.QueryAllApplicationsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.AllApplications(cmd.Context(), params)
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

func CmdShowApplication() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-application <application_address>",
		Short: "shows a application",
		Long: `Finds a staked application given its address.

Example:
$ pocketd q application show-application $(APP_ADDRESS) --node $(POCKET_NODE) --home $(POCKETD_HOME)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argAddress := args[0]

			params := &types.QueryGetApplicationRequest{
				Address: argAddress,
			}

			res, err := queryClient.Application(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
