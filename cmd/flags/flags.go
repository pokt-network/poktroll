package flags

import (
	"fmt"
	"strings"

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

	FlagNetwork      = "network"
	FlagNetworkUsage = "Sets the --node, --grpc-addr, and --chain-id flags (if applicable) based on the given network moniker (e.g. alpha, beta, main)"
	DefaultNetwork   = ""

	AlphaNetworkName = "alpha"
	BetaNetworkName  = "beta"
	MainNetworkName  = "main"
)

// TODO_IN_THIS_COMMIT: godoc..
func GetStringIfRegistered(cmd *cobra.Command, flagName string) (value string, isRegistered bool, err error) {
	flagValue, err := cmd.Flags().GetString(flagName)

	// Skip flags that are not defined on the current command.
	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("flag accessed but not defined: %s", flags.FlagNode)) {
		return "", false, nil
	}

	return flagValue, true, nil
}
