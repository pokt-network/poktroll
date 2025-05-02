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

// startCmd is the subcommand for running the relay miner (was root logic).
func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start --config <path>",
		Short: "Start a RelayMiner",
		Long: `Start a RelayMiner Process

A RelayMiner is the coprocessor that runs offchain to handle relays, provide a service and earn rewards. It:
- Handles incoming relay requests, validates, proxies, signs, and returns responses
- Hashes/computes relay difficulty; eligible relays are persisted for rewards
- Monitors block height, submits claim/proof messages as sessions are eligible
`,
		RunE: runRelayer,
	}
	// Custom flags
	cmd.Flags().StringVar(&flagRelayMinerConfig, "config", "", "The path to the relayminer config file")
	cmd.Flags().Bool(config.FlagQueryCaching, true, "Enable or disable onchain query caching")

	// Cosmos flags
	// TODO_TECHDEBT(#256): Remove unneeded cosmos flags.
	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().StringVar(&flagNodeRPCURL, cosmosflags.FlagNode, flags.OmittedDefaultFlagValue, "Register the default Cosmos node flag, which is needed to initialize the Cosmos query and tx contexts correctly. It can be used to override the `QueryNodeRPCURL` and `TxNodeRPCURL` fields in the config file if specified.")
	cmd.Flags().StringVar(&flagNodeGRPCURL, cosmosflags.FlagGRPC, flags.OmittedDefaultFlagValue, "Register the default Cosmos node grpc flag, which is needed to initialize the Cosmos query context with grpc correctly. It can be used to override the `QueryNodeGRPCURL` field in the config file if specified.")
	cmd.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options. It can be used to override the `QueryNodeGRPCInsecure` field in the config file if specified.")
	cmd.Flags().String(cosmosflags.FlagChainID, "pocket", "The network chain ID")
	cmd.Flags().StringVar(&flagLogLevel, cosmosflags.FlagLogLevel, "debug", "The logging level (debug|info|warn|error)")
	cmd.Flags().Float64(cosmosflags.FlagGasAdjustment, 1.7, "The adjustment factor to be multiplied by the gas estimate returned by the tx simulation")
	cmd.Flags().String(cosmosflags.FlagGasPrices, "1upokt", "Set the gas unit price in upokt")

	return cmd
}

// runRelayer starts the relay miner with the provided configuration and context.
func runRelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx() // Ensure context cancellation

	// Handle interrupt/kill signals asynchronously.
	signals.GoOnExitSignal(cancelCtx)

	configContent, err := os.ReadFile(flagRelayMinerConfig)
	if err != nil {
		return err
	}

	// TODO_TECHDEBT: Add logger level/output options to config.
	relayMinerConfig, err := relayerconfig.ParseRelayMinerConfigs(configContent)
	if err != nil {
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

	// Sets up dependencies:
	// - Miner
	// - EventsQueryClient
	// - BlockClient
	// - cosmosclient.Context
	// - TxFactory
	// - TxContext
	// - TxClient
	// - SupplierClient
	// - RelayerProxy
	// - RelayerSessionsManager
	deps, err := setupRelayerDependencies(ctx, cmd, relayMinerConfig)
	if err != nil {
		return err
	}

	relayMiner, err := relayer.NewRelayMiner(ctx, deps)
	if err != nil {
		return err
	}

	// Serve metrics if enabled.
	if relayMinerConfig.Metrics.Enabled {
		err = relayMiner.ServeMetrics(relayMinerConfig.Metrics.Addr)
		if err != nil {
			return fmt.Errorf("failed to start metrics endpoint: %w", err)
		}
	}

	queryCachingEnabled, err := cmd.Flags().GetBool(config.FlagQueryCaching)
	if err != nil {
		return fmt.Errorf("failed to get query caching flag: %w", err)
	}

	if queryCachingEnabled {
		logger.Info().Msg("query caching enabled")
	} else {
		logger.Info().Msg("query caching disabled")
	}

	if relayMinerConfig.Pprof.Enabled {
		err = relayMiner.ServePprof(ctx, relayMinerConfig.Pprof.Addr)
		if err != nil {
			return fmt.Errorf("failed to start pprof endpoint: %w", err)
		}
	}

	if relayMinerConfig.Ping.Enabled {
		if err := relayMiner.ServePing(ctx, "tcp", relayMinerConfig.Ping.Addr); err != nil {
			return fmt.Errorf("failed to start ping endpoint: %w", err)
		}
	}

	// Start the relay miner
	logger.Info().Msg("Starting relay miner...")
	if err := relayMiner.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start relay miner: %w", err)
	} else if errors.Is(err, http.ErrServerClosed) {
		logger.Info().Msg("Relay miner stopped; exiting")
	}
	return nil
}
