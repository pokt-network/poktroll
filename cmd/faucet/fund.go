package faucet

import (
	"fmt"
	"io"
	"net/http"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
)

// fundURLFmt is the canonical fund URL format for a given denom and recipient address.
// The placeholders are intended to be interpolated in the following order:
//   - baseURL: Fully-qualified URL to the faucet server (e.g. https://shannon-testnet-grove-faucet.beta.poktroll.com)
//   - denom: the denom to fund (e.g. upokt)
//   - recipientAddress: the recipient address to fund (e.g. pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4)
//
// Curl example:
// curl -X POST -H "Content-Type: application/json" https://shannon-testnet-grove-faucet.beta.poktroll.com/upokt/pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4
const fundURLFmt = "%s/%s/%s"

var faucetBaseURL string

func FundCmd() *cobra.Command {
	fundCmd := &cobra.Command{
		Use:   "fund [denom] [recipient key name or address]",
		Args:  cobra.ExactArgs(2),
		Short: "Request tokens of a given denom be sent to a recipient key name or address.",
		Long: `Request tokens of a given denom be sent to a recipient key name or address.

The faucet fund command sends a POST request to fund the account with the token denom as specified by RESTful path parameters.
Requests are send to the faucet server at the endpoint specified by --faucet-base-url flag.
The --network flag can also be used to set the faucet base URL by network name (e.g. --network=beta; see: --help).
The amount of tokens sent is a server-side configuration parameter.

// TODO_UP_NEXT(@bryanchriswhite): update docs URL once known.
For more information, see: https://dev.poktroll.com/operate/faucet`,
		Example: `# Funding mact denom by key name, using default faucet base URL
pocketd faucet fund mact app1

# Funding mact denom by address, using default faucet base URL
pocketd faucet fund mact pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4

# Funding upokt denom by key name, custom faucet base URL
pocketd faucet fund upokt app --base-url=http://localhost:8080

# Funding mact denom by key name, faucet base URL set by --network flag
pocketd faucet fund mact app --network=main`,
		RunE: runFund,
	}

	fundCmd.Flags().StringVar(&faucetBaseURL, flags.FlagFaucetBaseURL, flags.DefaultFaucetBaseURL, flags.FlagFaucetBaseURLUsage)

	return fundCmd
}

// runFund parses the recipient address sends a request to the faucet server for the given address and denom.
func runFund(cmd *cobra.Command, args []string) error {
	denom := args[0]
	recipientAddressStrOrKeyName := args[1]

	// Conventionally derive a cosmos-sdk client context from the cobra command
	clientCtx, err := cosmosclient.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	// Attempt to parse the first argument as an address first (no key name should be an address).
	recipientAddress, err := cosmostypes.AccAddressFromBech32(recipientAddressStrOrKeyName)
	if err != nil {
		// Attempt to retrieve the address from the keyring.
		// If the key name is not found, an error is returned.
		var record *keyring.Record
		record, err = clientCtx.Keyring.Key(recipientAddressStrOrKeyName)
		if err != nil {
			return err
		}

		recipientAddress, err = record.GetAddress()
		if err != nil {
			return err
		}
	}

	if err = sendFundRequest(denom, recipientAddress); err != nil {
		return err
	}

	logger.Logger.Info().
		Str("denom", denom).
		Str("recipient_address", recipientAddress.String()).
		Msg("Success")

	return nil
}

// sendFundRequest sends an HTTP GET request to the faucet server for the given recipient address and denom.
func sendFundRequest(denom string, recipientAddress cosmostypes.AccAddress) error {
	fundURL := getFundURL(denom, recipientAddress)

	logger.Logger.Debug().
		Str("fund_url", fundURL).
		Str("denom", denom).
		Str("recipient_address", recipientAddress.String()).
		Msg("sending fund request")

	httpRes, err := http.DefaultClient.Post(fundURL, "text/json", nil)
	if err != nil {
		return err
	}

	// Read the response body (JSON).
	bodyBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return err
	}
	defer func() {
		_ = httpRes.Body.Close()
	}()

	bodyStr := string(bodyBytes)

	switch httpRes.StatusCode {
	case http.StatusAccepted:
		logger.Logger.Info().Msg(bodyStr)
		return nil
	case http.StatusNotModified:
		logger.Logger.Error().Msg(bodyStr)
		return nil
	default:
	}

	return fmt.Errorf("unexpected response status code %d; body: %q", httpRes.StatusCode, bodyStr)
}

// getFundURL interpolates the baseURL, recipientAddress, and denom into the canonical fund URL for a given denom and recipient address.
func getFundURL(denom string, recipientAddress cosmostypes.AccAddress) string {
	return fmt.Sprintf(fundURLFmt, faucetBaseURL, denom, recipientAddress)
}
