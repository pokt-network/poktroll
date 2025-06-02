package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
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

	// Cosmos flags
	cosmosflags.AddTxFlagsToCmd(cmdStart)
	// Cosmos FlagDefaults
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagGRPC).Value.Set("localhost:9090")
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagGasPrices).Value.Set("1upokt")
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagGasAdjustment).Value.Set("1.7")
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagLogLevel).Value.Set("debug")
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagKeyringBackend).Value.Set("test")
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagGRPCInsecure).Value.Set("true")
	_ = cmdStart.Flags().Lookup(cosmosflags.FlagChainID).Value.Set("pocket")

	// Required flags
	_ = cmdStart.MarkFlagRequired("config")
	_ = cmdStart.MarkFlagRequired(cosmosflags.FlagNode)
	_ = cmdStart.MarkFlagRequired(cosmosflags.FlagGRPC)
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
