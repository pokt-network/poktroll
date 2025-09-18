package faucet

import (
	"errors"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/faucet"
)

const (
	// envPrefix is the viper env prefix. This prefix must be used when setting viper values via environment variables.
	// See: https://github.com/spf13/viper?tab=readme-ov-file#working-with-environment-variables
	envPrefix = "FAUCET"
)

var (
	// faucetCfg is used to configure the faucet server.
	// It is initialized in preRunServe.
	faucetCfg *faucet.Config

	// txClient is used by the faucet server to send transactions.
	// It is initialized in preRunServe.
	txClient client.TxClient

	// bankQueryClient is used by the faucet server to query for the existence and/or balance of accounts.
	// It is initialized in preRunServe.
	bankQueryClient banktypes.QueryClient
)

func FaucetCmd() *cobra.Command {
	faucetCmd := &cobra.Command{
		Use:   "faucet",
		Short: "Pocket Network faucet client and server CLI",
		Long: `Pocket Network faucet client and server CLI.

The faucet serve command exposes a configurable REST HTTP endpoint to the faucet client.
It uses the configured Pocket Network RPC endpoint, keyring, and signing key to send transactions.

The faucet fund command sends a POST request to fund the account with the token denom as specified by RESTful path parameters.
Requests are send to the faucet server at the endpoint specified by --faucet-base-url flag.
The --network flag can also be used to set the faucet base URL by network name (e.g. --network=beta; see: --help).`,
	}

	faucetCmd.AddCommand(FundCmd())
	faucetCmd.AddCommand(ServeCmd())

	faucetCmd.PersistentFlags().StringVar(&logger.LogLevel, cosmosflags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	faucetCmd.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	return faucetCmd
}

// Setup viper reads viper config values from the following sources in order of precedence (highest to lowest):
// 1. Explicit viper.Set() calls
// 2. Bound flags
// 3. Environment variables
// 4. Persistent config file(s)
// 5. Defaults
// See: https://github.com/spf13/viper?tab=readme-ov-file#why-viper
func setupViper(cmd *cobra.Command) error {
	// Set up the viper config (search paths, extension, etc.).
	if err := setViperConfig(); err != nil {
		return err
	}

	// Bind all viper values to environment variables prefixed with the envPrefix.
	// See: https://github.com/spf13/viper?tab=readme-ov-file#working-with-environment-variables
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	// Bind the listen address flag to the viper config.
	listenAddressFlag := cmd.Flags().Lookup(flags.FaucetListenAddress)
	if err := viper.BindPFlag("listen_address", listenAddressFlag); err != nil {
		return err
	}

	// Set default values.
	setViperDefaults()
	return nil
}

// setViperConfig first sets up the viper config search paths, file name, and file extension;
// then it attempts to load it.
func setViperConfig() error {
	// name of config file (without extension)
	viper.SetConfigName("faucet_config")
	viper.SetConfigType("yaml")

	// If the faucet config path is provided, use it instead of searching.
	if faucetConfigPath != flags.DefaultFaucetConfigPath {
		viper.SetConfigFile(faucetConfigPath)
	} else {
		// call multiple times to add many search paths
		viper.AddConfigPath("$HOME/.pocket")
		viper.AddConfigPath("$HOME/.poktroll")
		viper.AddConfigPath(".")
	}

	// Find and read the config file
	err := viper.ReadInConfig()
	switch {
	// It's okay if the config file doesn't exist.
	// Configuration MAY be done via environment variables instead.
	// We will rely on the faucet.Config validation later.
	case errors.As(err, &viper.ConfigFileNotFoundError{}):
		return nil
	default:
		return err
	}
}

// setViperDefaults sets default values for the following required viper config values:
// - listen_address
// - signing_key_name
// - supported_send_coins
// - create_accounts_only
func setViperDefaults() {
	viper.SetDefault("listen_address", "")
	viper.SetDefault("signing_key_name", "faucet")
	viper.SetDefault("supported_send_coins", "1mact")
	viper.SetDefault("create_accounts_only", "false")
}
