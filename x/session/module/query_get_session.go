package session

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/proto/types/session"
)

var _ = strconv.Itoa(0)

func CmdGetSession() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-session <application_address> <service_id> <block_height>",
		Short: "Query get-session",
		Long: `Query the session data for a specific (app, service, height) tuple.

This is a query operation that will not result in a state transition but simply gives a view into the chain state.

Example:
$ poktrolld q session get-session pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 svc1 42 --node $(POCKET_NODE) --home $(POKTROLLD_HOME)`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			appAddressString := args[0]
			serviceIdString := args[1]
			blockHeightString := "0" // 0 will default to latest height
			if len(args) == 3 {
				blockHeightString = args[2]
			}

			blockHeight, parseErr := strconv.ParseInt(blockHeightString, 10, 64)
			if parseErr != nil {
				return fmt.Errorf("couldn't convert block height to int: %s; (%v)", blockHeightString, parseErr)
			}

			getSessionReq := session.NewQueryGetSessionRequest(appAddressString, serviceIdString, blockHeight)
			if err := getSessionReq.ValidateBasic(); err != nil {
				return err
			}

			clientCtx, ctxErr := client.GetClientQueryContext(cmd)
			if ctxErr != nil {
				return ctxErr
			}
			queryClient := session.NewQueryClient(clientCtx)

			getSessionRes, sessionErr := queryClient.GetSession(cmd.Context(), getSessionReq)
			if sessionErr != nil {
				return sessionErr
			}

			return clientCtx.PrintProto(getSessionRes)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
