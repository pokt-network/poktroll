package flags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	// OmittedDefaultFlagValue is used whenever a flag is required but no reasonable default value can be provided.
	// In most cases, this forces the user to specify the flag value to avoid unintended behavior.
	OmittedDefaultFlagValue = "intentionally omitting default"

	// DEV_NOTE: use cosmosflags.FlagGRPC for the flag name.
	FlagGRPCUsage = "Register the default Cosmos node grpc flag, which is needed to initialize the Cosmos query context with grpc correctly. It can be used to override the `QueryNodeGRPCURL` field in the config file if specified."

	// DEV_NOTE: use cosmosflags.FlagGRPCInsecure for the flag name.
	FlagGRPCInsecureUsage = "Allow gRPC over insecure channels, if not the server MUST be TLS terminated"

	// DEV_NOTE: use cosmosflags.FlagLogLevelUsage for the flag name.
	FlagLogLevelUsage = "The logging level (debug|info|warn|error)"
	DefaultLogLevel   = "info"

	FlagLogOutput      = "log-output"
	FlagLogOutputUsage = "The logging output (<path>|'discard'|'stderr'); defaults to stdout ('-')"
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

	BooleanTrueValue  = "true"
	BooleanFalseValue = "false"

	// FlagQueryCaching is the flag name to enable or disable query caching.
	FlagQueryCaching        = "query-caching"
	FlagQueryCachingUsage   = "(Optional) Enable or disable onchain query caching"
	DefaultFlagQueryCaching = true

	// DefaultNodeRPCURL is the cosmos-sdk default --node flag value.
	// - Hard-coded in cosmos-sdk CLI
	// - Cannot be changed since registered by cosmos-sdk
	// - Used only for comparison, not flag registration
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.53.2/client/flags/flags.go#L108
	DefaultNodeRPCURL = "tcp://localhost:26657"
)

// GetFlagValueString returns the value of the flag with the given name.
// If the flag is not registered, an error is returned.
func GetFlagValueString(cmd *cobra.Command, flagName string) (string, error) {
	flag, err := GetFlag(cmd, flagName)
	if err != nil {
		return "", err
	}

	return flag.Value.String(), nil
}

// GetFlagBool returns the boolean value of the flag with the given name.
// Returns error if flag is not registered or has invalid boolean value.
func GetFlagBool(cmd *cobra.Command, flagName string) (bool, error) {
	flagValueString, err := GetFlagValueString(cmd, flagName)
	if err != nil {
		return false, err
	}

	switch flagValueString {
	case BooleanTrueValue:
		return true, nil
	case BooleanFalseValue:
		return false, nil
	default:
		return false, ErrFlagInvalidValue.Wrapf("expected 'true' or 'false', got: %s", flagValueString)
	}
}

// GetFlag returns the flag with the given name.
// If the flag is not registered, an error is returned.
func GetFlag(cmd *cobra.Command, flagName string) (*pflag.Flag, error) {
	flag := cmd.Flag(flagName)
	if flag == nil {
		return nil, ErrFlagNotRegistered.Wrapf("flag name: %s", flagName)
	}

	return flag, nil
}
