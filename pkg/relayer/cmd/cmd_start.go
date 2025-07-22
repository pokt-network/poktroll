package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
)

// startCmd returns the Cobra subcommand for running the relay miner.
//
// RelayMiner Responsibilities:
// - Handle incoming relay requests (validate, proxy, sign, return response)
// - Compute relay difficulty (determine reward eligible vs reward ineligible relays)
// - Monitor block height (submit claim/proof messages as sessions are eligible)
// - Cache various data
// - Rate limit incoming requests
func startCmd() *cobra.Command {
	cmdStart := &cobra.Command{
		Use:   "start --config <path-to-relay-miner-config-file> --chain-id <chain-id>",
		Short: "Start a RelayMiner",
		Long: `Start a RelayMiner Process.

A RelayMiner is an offchain coprocessor that provides a service.

RelayMiner Responsibilities:
- Handle incoming relay requests (validate, proxy, sign, return response)
- Compute relay difficulty (determine reward eligible vs reward ineligible relays)
- Monitor block height (submit claim/proof messages as sessions are eligible)
- Cache various data
- Rate limit incoming requests
`,
		RunE: runRelayer,
	}

	// Global logger flags
	// DEV_NOTE: Since the root command runs logger.PreRunESetup(), we need to ensure that the log level and output flags are registered on this subcommand.
	cmdStart.PersistentFlags().StringVar(&logger.LogLevel, cosmosflags.FlagLogLevel, "info", flags.FlagLogLevelUsage)
	cmdStart.PersistentFlags().StringVar(&logger.LogOutput, flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	// Custom flags
	cmdStart.Flags().StringVar(&relayMinerConfigPath, FlagConfig, DefaultFlagConfig, FlagConfigUsage)
	cmdStart.Flags().BoolVar(&flagQueryCaching, flags.FlagQueryCaching, flags.DefaultFlagQueryCaching, flags.FlagQueryCachingUsage)

	// Required cosmos-sdk CLI query flags.
	cmdStart.Flags().String(cosmosflags.FlagGRPC, flags.OmittedDefaultFlagValue, flags.FlagGRPCUsage)
	cmdStart.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, flags.FlagGRPCInsecureUsage)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(cmdStart)

	// Required flags
	_ = cmdStart.MarkFlagRequired(FlagConfig)
	// TODO_TECHDEBT(@olshansk): Consider making this part of the relay miner config file or erroring in a more user-friendly way.
	_ = cmdStart.MarkFlagRequired(cosmosflags.FlagChainID)

	return cmdStart
}

// runRelayer starts the relay miner with the provided configuration and context.
//
// Responsibilities:
// - Handle signal interruptions
// - Load and parse configuration
// - Set up logger and dependencies
// - Initialize and start the relay miner
func runRelayer(cmd *cobra.Command, _ []string) error {
	// --- Context setup and cancellation ---
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx() // Ensure context cancellation

	// Retrieve the logger from the command context.
	logger := polylog.Ctx(cmd.Context())

	// --- Signal handling ---
	signals.GoOnExitSignal(logger, cancelCtx)

	// Read relay miner config file
	configContent, err := os.ReadFile(relayMinerConfigPath)
	if err != nil {
		fmt.Printf("Could not read config file from: %s\n", relayMinerConfigPath)
		return err
	}

	// --- Print full-node configuration guidelines ---
	// Not using logger here to avoid multiple log entries and json formatting issues.
	fmt.Printf(`
❗RPC Full Node Configuration Guide ❗
====================================

🔧 When running multiple RelayMiners or Suppliers, adjust these settings
in your Full Node's config.toml file:

📐 Configuration Formulas:
-------------------------
🩺 Subscriptions
  - 'max_subscriptions_per_client' > 'total_suppliers' + 'total_relay_miners'
  - Each Supplier needs 1 subscription
  - Each RelayMiner needs 1 subscription

🔌 Connections:
  - 'max_open_connections' > 2 × 'total_relay_miners'
  - Each RelayMiner typically needs 2 connections

💡 Example Setup:
----------------
• RelayMiner 1: 2 Suppliers
• RelayMiner 2: 3 Suppliers
• RelayMiner 3: 1 Supplier

Totals:
- 'total_suppliers' = 6
- 'total_relay_miners' = 3

✅ Required config.toml settings:
'max_subscriptions_per_client' = 10  (must be > 6 + 3 = 9)
'max_open_connections' = 7           (must be > 2 × 3 = 6)
`)

	// --- Parse relay miner configuration ---
	// TODO_IMPROVE: Add logger level/output options to config.
	relayMinerConfig, err := relayerconfig.ParseRelayMinerConfigs(logger, configContent)
	if err != nil {
		fmt.Printf("Could not parse config file from: %s\n", relayMinerConfigPath)
		return err
	}

	// --- Log flag values ---
	if err = logFlagValues(logger, cmd); err != nil {
		logger.Error().Err(err).Msg("Could not read provided flags")
		return err
	}

	// Log query caching status
	if flagQueryCaching {
		logger.Info().Msg("query caching ENABLED")
	} else {
		logger.Info().Msg("query caching DISABLED")
	}

	// --- Set up dependencies for relay miner ---
	deps, err := setupRelayerDependencies(ctx, cmd, relayMinerConfig)
	if err != nil {
		logger.Error().Err(err).Msg("Could not setup dependencies")
		return err
	}

	// --- Initialize the relay miner ---
	relayMiner, err := relayer.NewRelayMiner(ctx, deps)
	if err != nil {
		logger.Error().Err(err).Msg("Could not initialize relay miner")
		return err
	}

	// --- Serve metrics endpoint if enabled ---
	if relayMinerConfig.Metrics.Enabled {
		err = relayMiner.ServeMetrics(relayMinerConfig.Metrics.Addr)
		if err != nil {
			logger.Error().Err(err).Msg("Could not start metrics endpoint")
			return err
		}
	}

	// --- Serve pprof endpoint if enabled ---
	if relayMinerConfig.Pprof.Enabled {
		err = relayMiner.ServePprof(ctx, relayMinerConfig.Pprof.Addr)
		if err != nil {
			logger.Error().Err(err).Msg("Could not start pprof endpoint")
			return err
		}
	}

	// --- Serve ping endpoint if enabled ---
	if relayMinerConfig.Ping.Enabled {
		if err = relayMiner.ServePing(ctx, "tcp", relayMinerConfig.Ping.Addr); err != nil {
			logger.Error().Err(err).Msg("Could not start ping endpoint")
			return err
		}
	}

	// --- Start the relay miner ---
	logger.Info().Msg("Starting relay miner...")

	err = relayMiner.Start(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error().Err(err).Msg("Could not start relay miner")
		return err
	}
	if errors.Is(err, http.ErrServerClosed) {
		logger.Info().Msg("Relay miner stopped; exiting")
		return err
	}

	return nil
}

// logFlagValues logs the flags provided to the command.
// It logs the chain ID, version, home directory, keyring backend, and gRPC insecure flag.
// This is useful for debugging and ensuring the correct configuration is used.
func logFlagValues(logger polylog.Logger, cmd *cobra.Command) error {
	clientCtx := client.GetClientContextFromCmd(cmd)

	logger.Info().Msgf(
		"Config in use: chain_id: %s, version: %s, home: %s, keyring_backend: %s, keyring_dir: %s",
		clientCtx.ChainID,
		version.NewInfo().Version,
		clientCtx.HomeDir,
		clientCtx.Keyring.Backend(),
		clientCtx.KeyringDir,
	)

	return nil
}
