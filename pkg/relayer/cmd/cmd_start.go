package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
)

// startCmd returns the Cobra subcommand for running the relay miner.
//
// Responsibilities of a RelayMiner include:
// - Handling incoming relay requests: Validate, proxy, sign, return response, etc.
// - Computing relay difficulty: Determining reward eligible vs reward ineligible relays
// - Monitoring block height: Submitting claim/proof messages as sessions are eligible
// - Caching of various sorts
// - Rate limiting incoming requests
func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start --config <path-to-relay-miner-config-file>",
		Short: "Start a RelayMiner",
		Long: `Start a RelayMiner Process.

A RelayMiner is an offchain coprocessor that provides a service.

Responsibilities:
- Handle incoming relay requests: Validate, proxy, sign, return response, etc.
- Compute relay difficulty: Determine reward eligible vs reward ineligible relays
- Monitor block height: Submit claim/proof messages as sessions are eligible
- Cache various data
- Rate limit incoming requests
`,
		RunE: runRelayer,
	}

	// Custom flags
	cmd.Flags().StringVar(&flagRelayMinerConfig, "config", "", "(Required) The path to the relayminer config file")
	cmd.Flags().BoolVar(&flagQueryCaching, config.FlagQueryCaching, true, "(Optional) Enable or disable onchain query caching")

	// Cosmos flags
	cmd.Flags().StringVar(&flagNodeRPCURL, cosmosflags.FlagNode, flags.OmittedDefaultFlagValue, "Register the default Cosmos node flag, which is needed to initialize the Cosmos query and tx contexts correctly. It can be used to override the `QueryNodeRPCURL` and `TxNodeRPCURL` fields in the config file if specified.")
	cmd.Flags().StringVar(&flagNodeGRPCURL, cosmosflags.FlagGRPC, flags.OmittedDefaultFlagValue, "Register the default Cosmos node grpc flag, which is needed to initialize the Cosmos query context with grpc correctly. It can be used to override the `QueryNodeGRPCURL` field in the config file if specified.")
	cmd.Flags().StringVar(&flagLogLevel, cosmosflags.FlagLogLevel, "debug", "The logging level (debug|info|warn|error)")
	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options. It can be used to override the `QueryNodeGRPCInsecure` field in the config file if specified.")
	cmd.Flags().String(cosmosflags.FlagChainID, "pocket", "The network chain ID")
	cmd.Flags().Float64(cosmosflags.FlagGasAdjustment, 1.7, "The adjustment factor to be multiplied by the gas estimate returned by the tx simulation")
	cmd.Flags().String(cosmosflags.FlagGasPrices, "1upokt", "Set the gas unit price in upokt")

	return cmd
}

// TODO_TECHDEBT(@olshansk): Move flags into the startCmd function above.
// This is necessary for backwards compatibility with old config files so "start"
// is the default if not subcommand is specified.
func startCmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flagRelayMinerConfig, "config", "", "(Required) The path to the relayminer config file")
	cmd.PersistentFlags().BoolVar(&flagQueryCaching, config.FlagQueryCaching, true, "(Optional) Enable or disable onchain query caching")
}

// runRelayer starts the relay miner with the provided configuration and context.
//
// - Handles signal interruptions
// - Loads and parses configuration
// - Sets up logger and dependencies
// - Initializes and starts the relay miner
func runRelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx() // Ensure context cancellation

	// Handle interrupt/kill signals asynchronously.
	signals.GoOnExitSignal(cancelCtx)

	configContent, err := os.ReadFile(flagRelayMinerConfig)
	if err != nil {
		fmt.Printf("Could not read config file from: %s\n", flagRelayMinerConfig)
		return err
	}

	// TODO_IMPROVE: Add logger level/output options to config.
	relayMinerConfig, err := relayerconfig.ParseRelayMinerConfigs(configContent)
	if err != nil {
		fmt.Printf("Could not parse config file from: %s\n", flagRelayMinerConfig)
		return err
	}

	// TODO_TECHDEBT: Populate logger from config (ideally, from viper).
	loggerOpts := []polylog.LoggerOption{
		polyzero.WithLevel(polyzero.ParseLevel(flagLogLevel)),
		polyzero.WithOutput(os.Stderr),
	}

	// Construct logger and associate with command context.
	logger := polyzero.NewLogger(loggerOpts...)
	ctx = logger.WithContext(ctx)
	cmd.SetContext(ctx)

	if flagQueryCaching {
		logger.Info().Msg("query caching ENABLED")
	} else {
		logger.Info().Msg("query caching DISABLED")
	}

	// Sets up dependencies
	deps, err := setupRelayerDependencies(ctx, cmd, relayMinerConfig)
	if err != nil {
		fmt.Printf("Could not setup dependencies: %v\n", err)
		return err
	}

	// Initialize the relay miner.
	relayMiner, err := relayer.NewRelayMiner(ctx, deps)
	if err != nil {
		fmt.Printf("Could not initialize relay miner: %v\n", err)
		return err
	}

	// Serve metrics if enabled.
	if relayMinerConfig.Metrics.Enabled {
		err = relayMiner.ServeMetrics(relayMinerConfig.Metrics.Addr)
		if err != nil {
			fmt.Printf("Could not start metrics endpoint: %v\n", err)
			return err
		}
	}

	// Serve pprof if enabled.
	if relayMinerConfig.Pprof.Enabled {
		err = relayMiner.ServePprof(ctx, relayMinerConfig.Pprof.Addr)
		if err != nil {
			fmt.Printf("Could not start pprof endpoint: %v\n", err)
			return err
		}
	}

	// Serve ping if enabled.
	if relayMinerConfig.Ping.Enabled {
		if err = relayMiner.ServePing(ctx, "tcp", relayMinerConfig.Ping.Addr); err != nil {
			fmt.Printf("Could not start ping endpoint: %v\n", err)
			return err
		}
	}

	// Start the relay miner
	logger.Info().Msg("Starting relay miner...")
	err = relayMiner.Start(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Could not start relay miner: %v\n", err)
		return err
	}
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Relay miner stopped; exiting\n")
		return err
	}
	return nil
}
