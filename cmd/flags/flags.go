package flags

import (
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	// OmittedDefaultFlagValue is used whenever a flag is required but no reasonable default value can be provided.
	// In most cases, this forces the user to specify the flag value to avoid unintended behavior.
	OmittedDefaultFlagValue = "intentionally omitting default"

	FlagLogLevel      = "log-level"
	FlagLogLevelUsage = "The logging level (debug|info|warn|error)"
	DefaultLogLevel   = "info"

	FlagLogOutput      = "log-output"
	FlagLogOutputUsage = "The logging output (file path); defaults to stdout"
	DefaultLogOutput   = "-"

	FlagPassphrase      = "passphrase"
	FlagPassphraseShort = "p"
	FlagPassphraseUsage = "the passphrase used to decrypt the exported Morse key file for signing; the user will be prompted if empty (UNLESS --no-passphrase is used)"

	FlagNoPassphrase      = "no-passphrase"
	FlagNoPassphraseUsage = "attempt to use an empty passphrase to decrypt the exported Morse key file for signing"

	FlagInputFile      = "input-file"
	FlagInputFileUsage = "An absolute or relative path to an input file that can be used to read data from. This will not be overwritten."

	FlagOutputFile      = "output-file"
	FlagOutputFileUsage = "An absolute or relative path to an output file that can be used to write data to. Caution that this file may be updated or overwritten if it already exists."

	FlagNetwork      = "network"
	FlagNetworkUsage = "Sets the --chain-id, --node, and --grpc-addr flags (if applicable) based on the given network moniker (e.g. local, alpha, beta, main)"
	DefaultNetwork   = ""

	FlagFaucetBaseURL      = "base-url"
	FlagFaucetBaseURLUsage = "The base URL of the Pocket Network Faucet"
	// TODO_UP_NEXT(@bryanchriswhite): Update to the MainNet URL once available.
	DefaultFaucetBaseURL = "https://shannon-testnet-grove-faucet.beta.poktroll.com"

	FaucetConfigPath = "faucet-config-path"
	// TODO_UP_NEXT(@bryanchriswhite): explicitly set config.
	FaucetConfigPathUsage   = "Path to the faucet config yaml file ($HOME/.{pocket,poktroll} and PWD are searched by default)"
	DefaultFaucetConfigPath = ""

	FaucetListenAddress        = "listen-address"
	FaucetListenAddressUsage   = "The listen address of the Pocket Network Faucet in the form of host:port"
	DefaultFaucetListenAddress = "0.0.0.0:8080"

	LocalNetworkName = "local"
	AlphaNetworkName = "alpha"
	BetaNetworkName  = "beta"
	MainNetworkName  = "main"
)

// flagHelpUpdates maps flag names to their updated help text using upokt denomination
var flagHelpUpdates = map[string]string{
	flags.FlagFees:      "Fees to pay along with transaction; eg: 10upokt",
	flags.FlagGasPrices: "Gas prices in decimal format to determine the transaction fee (e.g. 0.1upokt)",
	flags.FlagGas:       "gas limit to set per-transaction; set to \"auto\" to calculate sufficient gas automatically. Note: \"auto\" option doesn't always report accurate results. Set a valid coin value to adjust the result. Can be used instead of \"fees\". (default 200upokt)",
}

// updateFlagHelpText updates the help text for flags that reference "uatom" to use "upokt"
func updateFlagHelpText(cmd *cobra.Command) {
	for flagName, helpText := range flagHelpUpdates {
		if flag := cmd.Flags().Lookup(flagName); flag != nil {
			flag.Usage = helpText
		}
	}
}

// AddTxFlagsToCmd adds the standard transaction flags to the given command
// and updates the help text to use "upokt" instead of "uatom" denomination.
func AddTxFlagsToCmd(cmd *cobra.Command) {
	// Add the standard cosmos-sdk transaction flags
	flags.AddTxFlagsToCmd(cmd)

	// Update the help text for flags that reference "uatom" to use "upokt"
	updateFlagHelpText(cmd)
}
