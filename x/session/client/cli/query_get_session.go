package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"pocket/x/session/types"
)

var _ = strconv.Itoa(0)

func CmdGetSession() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-session [application address] [service ID] [block height]",
		Short: "Query get-session",
		Long: `Query the session data for a specific (app, service, height) tuple. This is a query operation
that will not result in a state transition but simply gives a view into the chain state.

Example:
$ pocketd --home=$(POCKETD_HOME) q session get-session pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 svc1 42 --node $(POCKET_NODE)`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			appAddressString := args[0]
			serviceIdString := args[1]
			blockHeightString := args[2]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			blockHeight, err := strconv.ParseInt(blockHeightString, 10, 64)
			if err != nil {
				return fmt.Errorf("couldn't convert block height to int: %s; (%v)", blockHeightString, err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			getSessionReq := types.NewQueryGetSessionRequest(appAddressString, serviceIdString, blockHeight)
			if err := getSessionReq.ValidateBasic(); err != nil {
				return err
			}

			getSessionRes, err := queryClient.GetSession(cmd.Context(), getSessionReq)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(getSessionRes)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
