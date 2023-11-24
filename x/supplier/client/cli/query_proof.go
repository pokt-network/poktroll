package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/spf13/cobra"
)

func CmdListProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-proofs",
		Short: "list all proofs",
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

			params := &types.QueryAllProofsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.AllProofs(cmd.Context(), params)
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

// TODO_UPNEXT(@Olshansk): Remove the dependency on index which was part of the default scaffolding behaviour
func CmdShowProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-proof <index>",
		Short: "shows a proof",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetProofRequest{
				Index: argIndex,
			}

			res, err := queryClient.Proof(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
