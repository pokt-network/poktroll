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
		Use:   "get-session <application_address> <service_id> [block_height]",
		Short: "Query get-session",
		Long: `Query the session data for a specific (app, service, height) tuple.

[block_height] is optional. If unspecified, or set to 0, it defaults to the latest height of the node being queried.

This is a query operation that will not result in a state transition but simply gives a view into the chain state.

Example:
$ pocketd --home=$(POCKETD_HOME) q session get-session pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 svc1 42 --node $(POCKET_NODE)`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			appAddressString := args[0]
			serviceIdString := args[1]
			blockHeightString := "0" // 0 will default to latest height
			if len(args) == 3 {
				blockHeightString = args[2]
			}

			blockHeight, err := strconv.ParseInt(blockHeightString, 10, 64)
			if err != nil {
				return fmt.Errorf("couldn't convert block height to int: %s; (%v)", blockHeightString, err)
			}

			getSessionReq := types.NewQueryGetSessionRequest(appAddressString, serviceIdString, blockHeight)
			if err := getSessionReq.ValidateBasic(); err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

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
