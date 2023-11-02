package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/session/types"
)

var _ = strconv.Itoa(0)

// TODO(@Olshansk): Implement the CLI component of `GetSession`.
func CmdGetSession() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-session",
		Short: "Query get-session",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryGetSessionRequest{}

			res, err := queryClient.GetSession(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
