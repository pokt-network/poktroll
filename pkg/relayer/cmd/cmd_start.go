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

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
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

	// Custom flags
	cmdStart.Flags().StringVar(&flagRelayMinerConfig, "config", "", "(Required) The path to the relayminer config file")
	cmdStart.Flags().BoolVar(&flagQueryCaching, config.FlagQueryCaching, true, "(Optional) Enable or disable onchain query caching")

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(cmdStart)

	// Required flags
	_ = cmdStart.MarkFlagRequired("config")
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
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx() // Ensure context cancellation

	// Set up logger options
	// TODO_TECHDEBT: Populate logger from config (ideally, from viper).
	loggerOpts := []polylog.LoggerOption{
		polyzero.WithLevel(polyzero.ParseLevel(flagLogLevel)),
		polyzero.WithOutput(os.Stderr),
	}

	// Construct logger and associate with command context
	logger := polyzero.NewLogger(loggerOpts...)
	ctx = logger.WithContext(ctx)
	cmd.SetContext(ctx)

	// Handle interrupt/kill signals asynchronously
	signals.GoOnExitSignal(cancelCtx)

	// Read relay miner config file
	configContent, err := os.ReadFile(flagRelayMinerConfig)
	if err != nil {
		fmt.Printf("Could not read config file from: %s\n", flagRelayMinerConfig)
		return err
	}

	// Parse relay miner configuration
	// TODO_IMPROVE: Add logger level/output options to config.
	relayMinerConfig, err := relayerconfig.ParseRelayMinerConfigs(configContent)
	if err != nil {
		fmt.Printf("Could not parse config file from: %s\n", flagRelayMinerConfig)
		return err
	}

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

	// Set up dependencies for relay miner
	deps, err := setupRelayerDependencies(ctx, cmd, relayMinerConfig)
	if err != nil {
		logger.Error().Err(err).Msg("Could not setup dependencies")
		return err
	}

	// Initialize the relay miner
	relayMiner, err := relayer.NewRelayMiner(ctx, deps)
	if err != nil {
		logger.Error().Err(err).Msg("Could not initialize relay miner")
		return err
	}

	// Serve metrics endpoint if enabled
	if relayMinerConfig.Metrics.Enabled {
		err = relayMiner.ServeMetrics(relayMinerConfig.Metrics.Addr)
		if err != nil {
			logger.Error().Err(err).Msg("Could not start metrics endpoint")
			return err
		}
	}

	// Serve pprof endpoint if enabled
	if relayMinerConfig.Pprof.Enabled {
		err = relayMiner.ServePprof(ctx, relayMinerConfig.Pprof.Addr)
		if err != nil {
			logger.Error().Err(err).Msg("Could not start pprof endpoint")
			return err
		}
	}

	// Serve ping endpoint if enabled
	if relayMinerConfig.Ping.Enabled {
		if err = relayMiner.ServePing(ctx, "tcp", relayMinerConfig.Ping.Addr); err != nil {
			logger.Error().Err(err).Msg("Could not start ping endpoint")
			return err
		}
	}

	// Start the relay miner
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
		"Config in use: chain_id: %s, version: %s, home: %s, keyring_backend: %s, keyring_dir: %s, grpc_insecure: %s",
		clientCtx.ChainID,
		version.NewInfo().Version,
		clientCtx.HomeDir,
		clientCtx.Keyring.Backend(),
		clientCtx.KeyringDir,
		cmd.Flag(cosmosflags.FlagGRPCInsecure).Value.String(),
	)

	return nil
}
