package faucet

import (
	"context"
	"strings"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
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
		Short: "Pocket Network Faucet",
		Long:  `Pocket Network Faucet`,
	}

	serveCmd := &cobra.Command{
		Use: "serve",
		// TODO_IN_THIS_COMMIT: ...
		//Short:,
		//Long:,
		//Example:,
		PreRunE: preRunServe,
		RunE:    runServe,
	}

	fundCmd := &cobra.Command{
		Use: "fund",
		// TODO_IN_THIS_COMMIT: ...
		//Short:,
		//Long:,
		//Example:,
		//RunE: runFund,
	}

	faucetCmd.AddCommand(fundCmd)
	faucetCmd.AddCommand(serveCmd)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(serveCmd)

	faucetCmd.PersistentFlags().StringVar(&logger.LogLevel, flags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	faucetCmd.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	if err := setupViper(); err != nil {
		panic(err)
	}

	return faucetCmd
}

// Setup viper reads viper config values from the following sources in order of precendence (highest to lowest):
// 1. Explicit viper.Set() calls
// 2. Bound flags (not currently in use)
// 3. Environment variables
// 4. Persistent config file(s)
// 5. Defaults
// See: https://github.com/spf13/viper?tab=readme-ov-file#why-viper
func setupViper() error {
	// Set up the viper config (search paths, extension, etc.).
	if err := setViperConfig(); err != nil {
		return err
	}

	// Bind all viper values to environment variables prefixed with the envPrefix.
	// See: https://github.com/spf13/viper?tab=readme-ov-file#working-with-environment-variables
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

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

	// call multiple times to add many search paths
	viper.AddConfigPath("$HOME/.pocket")
	viper.AddConfigPath("$HOME/.poktroll")
	viper.AddConfigPath(".")

	// Find and read the config file
	return viper.ReadInConfig()
}

// setViperDefaults sets default values for the following required viper config values:
// - signing_key_name
// - send_tokens
// - listen_address
func setViperDefaults() {
	viper.SetDefault("signing_key_name", "faucet")
	viper.SetDefault("send_tokens", "1mact")
	viper.SetDefault("listen_address", "")
}

// preRunServe performs the following setup steps:
// - retrieves the cosmos-sdk client context from the cobra command
// - unmarshals viper config values into a new FaucetConfig struct
// - constructs a tx client for use in the faucet
// - constructs a bank query client for use in the faucet
func preRunServe(cmd *cobra.Command, _ []string) (err error) {
	// Conventionally derive a cosmos-sdk client context from the cobra command
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	faucetCfg, err = parseFaucetConfigFromViper(clientCtx)
	if err != nil {
		return err
	}

	// Construct a tx client.
	var txClientOpts []client.TxClientOption
	signingKeyOpt := tx.WithSigningKeyName(faucetCfg.SigningKeyName)
	if err = cosmosclient.SetCmdClientContext(cmd, clientCtx); err != nil {
		return err
	}
	txClientOpts = append(txClientOpts, signingKeyOpt)

	unorderedOpt := tx.WithUnordered()
	txClientOpts = append(txClientOpts, unorderedOpt)

	// Construct the tx client.
	txClient, err = flags.GetTxClientFromFlags(cmd.Context(), cmd, txClientOpts...)
	if err != nil {
		return err
	}

	// Construct the account query client and dependencies.
	bankQueryClient = banktypes.NewQueryClient(clientCtx)

	return nil
}

// runServe starts the faucet server.
func runServe(cmd *cobra.Command, _ []string) error {
	logger.Logger.Info().
		Str("signing_key_name", faucetCfg.SigningKeyName).
		Str("send_coins", strings.Join(faucetCfg.SupportedSendCoins, ",")).
		Msgf("Listening on %s", faucetCfg.ListenAddress)

	cmdContext, cmdCancel := context.WithCancel(cmd.Context())
	cmd.SetContext(cmdContext)

	signals.GoOnExitSignal(func() {
		logger.Logger.Info().Msg("Shutting down...")
		cmdCancel()
	})

	faucetSrv, err := faucet.NewFaucetServer(
		cmdContext,
		faucet.WithConfig(faucetCfg),
		faucet.WithTxClient(txClient),
		faucet.WithBankQueryClient(bankQueryClient),
	)
	if err != nil {
		return err
	}

	return faucetSrv.Serve(cmdContext)
}

// parseFaucetConfigFromViper performs the following steps:
// - unmarshal the current viper config values into a new FaucetConfig struct
// - validate the resulting faucet config
// - load the faucet config's signing key (from the keyring)
func parseFaucetConfigFromViper(clientCtx cosmosclient.Context) (*faucet.Config, error) {
	config := new(faucet.Config)
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	// Ensure the faucet config is valid.
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Ensure the signing key and related config fields are ready to be used.
	if err := config.LoadSigningKey(clientCtx); err != nil {
		return nil, err
	}

	return config, nil
}
