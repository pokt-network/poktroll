package faucet

import (
	"fmt"
	"net/http"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/logger"
)

// TODO_IN_THIS_COMMIT: godoc...
const fundURLFmt = "%s/%s/%s"

var faucetBaseURL string

func FundCmd() *cobra.Command {
	fundCmd := &cobra.Command{
		Use:  "fund [recipient address] [denom]",
		Args: cobra.ExactArgs(2),
		// TODO_IN_THIS_COMMIT: ...
		//Short:,
		//Long:,
		Example: `pocketd faucet fund pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 upokt
pocketd faucet fund pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 mact`,
		//PreRunE: preRunFund,
		RunE: runFund,
	}

	networkflag
	fundCmd.Flags().StringVar(&faucetBaseURL, flags.FlagFaucetBaseURL, flags.DefaultFaucetBaseURL, flags.FlagFaucetBaseURLUsage)

	return fundCmd
}

//// TODO_IN_THIS_COMMIT: godoc...
//func preRunFund(cmd *cobra.Command, _ []string) error {
//
//}

// TODO_IN_THIS_COMMIT: godoc...
func runFund(cmd *cobra.Command, args []string) error {
	recipientAddressStr := args[0]
	denom := args[1]

	recipientAddress, err := cosmostypes.AccAddressFromBech32(recipientAddressStr)
	if err != nil {
		return err
	}

	logger.Logger.Info().
		Str("recipient_address", recipientAddressStr).
		Str("denom", denom).
		Msg("Funding recipient address...")

	sendFundRequest(recipientAddress, denom)

	return nil
}

// TODO_IN_THIS_COMMIT: godoc...
func sendFundRequest(recipientAddress cosmostypes.AccAddress, denom string) {
	fundURL := fmt.Sprintf(fundURLFmt, recipientAddress, denom)

	http.DefaultClient.Get(fundURL)
}
