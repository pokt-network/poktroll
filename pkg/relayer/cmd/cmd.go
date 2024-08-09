package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
)

// We're `explicitly omitting default` so the relayer crashes if these aren't specified.
const omittedDefaultFlagValue = "explicitly omitting default"

// TODO_CONSIDERATION: Consider moving all flags defined in `/pkg` to a `flags.go` file.
var (
	// flagRelayMinerConfig is the variable containing the relay miner config filepath
	// sourced from the `--config` flag.
	flagRelayMinerConfig string
	// flagNodeRPCURL is the variable containing the Cosmos node RPC URL flag value.
	flagNodeRPCURL string
	// flagNodeGRPCURL is the variable containing the Cosmos node GRPC URL flag value.
	flagNodeGRPCURL string
)

// RelayerCmd returns the Cobra command for running the relay miner.
func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayminer",
		Short: "Start a RelayMiner",
		Long: `Run a RelayMiner. A RelayMiner is the off-chain complementary
middleware that handles incoming requests for all the services a Supplier staked
for on-chain.

Relay requests received by the relay servers are validated and proxied to their
respective service endpoints, maintained by the relayer off-chain. The responses
are then signed and sent back to the requesting application.

For each successfully served relay, the miner will hash and compare its difficulty
against an on-chain threshold. If the difficulty is sufficient, it is applicable
to relay volume and therefore rewards. Such relays are inserted into and persisted
via an SMT KV store. The miner will monitor the current block height and periodically
submit claim and proof messages according to the protocol as sessions become eligible
for such operations.`,
		RunE: runRelayer,
	}
	// Custom flags
	cmd.Flags().StringVar(&flagRelayMinerConfig, "config", "", "The path to the relayminer config file")

	// Cosmos flags
	// TODO_TECHDEBT(#256): Remove unneeded cosmos flags.
	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().StringVar(&flagNodeRPCURL, cosmosflags.FlagNode, omittedDefaultFlagValue, "Register the default Cosmos node flag, which is needed to initialize the Cosmos query and tx contexts correctly. It can be used to override the `QueryNodeRPCURL` and `TxNodeRPCURL` fields in the config file if specified.")
	cmd.Flags().StringVar(&flagNodeGRPCURL, cosmosflags.FlagGRPC, omittedDefaultFlagValue, "Register the default Cosmos node grpc flag, which is needed to initialize the Cosmos query context with grpc correctly. It can be used to override the `QueryNodeGRPCURL` field in the config file if specified.")
	cmd.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options. It can be used to override the `QueryNodeGRPCInsecure` field in the config file if specified.")
	cmd.Flags().String(cosmosflags.FlagChainID, "poktroll", "The network chain ID")

	return cmd
}

func runRelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	// Ensure context cancellation.
	defer cancelCtx()

	// Handle interrupt and kill signals asynchronously.
	signals.GoOnExitSignal(cancelCtx)

	configContent, err := os.ReadFile(flagRelayMinerConfig)
	if err != nil {
		return err
	}

	// TODO_TECHDEBT: add logger level and output options to the config.
	relayMinerConfig, err := relayerconfig.ParseRelayMinerConfigs(configContent)
	if err != nil {
		return err
	}

	// TODO_TECHDEBT: populate logger from the config (ideally, from viper).
	loggerOpts := []polylog.LoggerOption{
		polyzero.WithLevel(polyzero.DebugLevel),
		polyzero.WithOutput(os.Stderr),
	}

	// Construct a logger and associate it with the command context.
	logger := polyzero.NewLogger(loggerOpts...)
	ctx = logger.WithContext(ctx)
	cmd.SetContext(ctx)

	// Sets up the following dependencies:
	// Miner, EventsQueryClient, BlockClient, cosmosclient.Context, TxFactory,
	// TxContext, TxClient, SupplierClient, RelayerProxy, RelayerSessionsManager.
	deps, err := setupRelayerDependencies(ctx, cmd, relayMinerConfig)
	if err != nil {
		return err
	}

	relayMiner, err := relayer.NewRelayMiner(ctx, deps)
	if err != nil {
		return err
	}

	// Serve metrics.
	if relayMinerConfig.Metrics.Enabled {
		err = relayMiner.ServeMetrics(relayMinerConfig.Metrics.Addr)
		if err != nil {
			return fmt.Errorf("failed to start metrics endpoint: %w", err)
		}
	}

	if relayMinerConfig.Pprof.Enabled {
		err = relayMiner.ServePprof(ctx, relayMinerConfig.Pprof.Addr)
		if err != nil {
			return fmt.Errorf("failed to start pprof endpoint: %w", err)
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

// setupRelayerDependencies sets up all the dependencies the relay miner needs
// to run by building the dependency tree from the leaves up, incrementally
// supplying each component to an accumulating depinject.Config:
// Miner, EventsQueryClient, BlockClient, cosmosclient.Context, TxFactory, TxContext,
// TxClient, SupplierClient, RelayerProxy, RelayerSessionsManager.
func setupRelayerDependencies(
	ctx context.Context,
	cmd *cobra.Command,
	relayMinerConfig *relayerconfig.RelayMinerConfig,
) (deps depinject.Config, err error) {
	queryNodeRPCUrl := relayMinerConfig.PocketNode.QueryNodeRPCUrl
	queryNodeGRPCUrl := relayMinerConfig.PocketNode.QueryNodeGRPCUrl
	txNodeRPCUrl := relayMinerConfig.PocketNode.TxNodeRPCUrl

	// Override the config file's `QueryNodeGRPCUrl` fields
	// with the `--grpc-addr` flag if it was specified.
	// TODO(#223) Remove this check once viper is used as SoT for overridable config values.
	if flagNodeGRPCURL != omittedDefaultFlagValue {
		parsedFlagNodeGRPCUrl, err := url.Parse(flagNodeGRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grpc query URL: %w", err)
		}
		queryNodeGRPCUrl = parsedFlagNodeGRPCUrl
	}

	// Override the config file's `QueryNodeUrl` and `txNodeRPCUrl` fields
	// with the `--node` flag if it was specified.
	// TODO(#223) Remove this check once viper is used as SoT for overridable config values.
	if flagNodeRPCURL != omittedDefaultFlagValue {
		parsedFlagNodeRPCUrl, err := url.Parse(flagNodeRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rpc query URL: %w", err)
		}
		queryNodeRPCUrl = parsedFlagNodeRPCUrl
		txNodeRPCUrl = parsedFlagNodeRPCUrl
	}

	signingKeyNames := uniqueSigningKeyNames(relayMinerConfig)
	servicesConfigMap := relayMinerConfig.Servers
	smtStorePath := relayMinerConfig.SmtStorePath

	supplierFuncs := []config.SupplierFn{
		config.NewSupplyLoggerFromCtx(ctx),
		config.NewSupplyEventsQueryClientFn(queryNodeRPCUrl),              // leaf
		config.NewSupplyBlockQueryClientFn(queryNodeRPCUrl),               // leaf
		config.NewSupplyBlockClientFn(queryNodeRPCUrl),                    // leaf
		config.NewSupplyQueryClientContextFn(queryNodeGRPCUrl),            // leaf
		config.NewSupplyTxClientContextFn(queryNodeGRPCUrl, txNodeRPCUrl), // leaf
		config.NewSupplyDelegationClientFn(),                              // leaf
		config.NewSupplySharedQueryClientFn(),                             // leaf
		config.NewSupplyProofQueryClientFn(),
		supplyMiner,
		config.NewSupplyAccountQuerierFn(),
		config.NewSupplyApplicationQuerierFn(),
		config.NewSupplySupplierQuerierFn(),
		config.NewSupplySessionQuerierFn(),
		config.NewSupplyRingCacheFn(),
		supplyTxFactory,
		supplyTxContext,
		config.NewSupplySupplierClientsFn(signingKeyNames),
		newSupplyRelayerProxyFn(signingKeyNames, servicesConfigMap),
		newSupplyRelayerSessionsManagerFn(smtStorePath),
	}

	return config.SupplyConfig(ctx, cmd, supplierFuncs)
}

// supplyMiner constructs a Miner instance and returns a new depinject.Config
// which is supplied with the given deps and the new Miner.
func supplyMiner(
	_ context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	mnr, err := miner.NewMiner(deps)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(mnr)), nil
}

// supplyTxFactory constructs a cosmostx.Factory instance and returns a new
// depinject.Config which is supplied with the given deps and the new
// cosmostx.Factory.
func supplyTxFactory(
	_ context.Context,
	deps depinject.Config,
	cmd *cobra.Command,
) (depinject.Config, error) {
	var txClientCtx txtypes.Context
	if err := depinject.Inject(deps, &txClientCtx); err != nil {
		return nil, err
	}

	clientCtx := cosmosclient.Context(txClientCtx)
	clientFactory, err := cosmostx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(clientFactory)), nil
}

func supplyTxContext(
	_ context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	txContext, err := tx.NewTxContext(deps)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(txContext)), nil
}

// newSupplyRelayerProxyFn returns a function which constructs a
// RelayerProxy instance and returns a new depinject.Config which
// is supplied with the given deps and the new RelayerProxy.
func newSupplyRelayerProxyFn(
	signingKeyNames []string,
	servicesConfigMap map[string]*relayerconfig.RelayMinerServerConfig,
) config.SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayerProxy, err := proxy.NewRelayerProxy(
			deps,
			proxy.WithSigningKeyNames(signingKeyNames),
			proxy.WithServicesConfigMap(servicesConfigMap),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(relayerProxy)), nil
	}
}

// newSupplyRelayerSessionsManagerFn returns a function which constructs a
// RelayerSessionsManager instance and returns a new depinject.Config which
// is supplied with the given deps and the new RelayerSessionsManager.
func newSupplyRelayerSessionsManagerFn(smtStorePath string) config.SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayerSessionsManager, err := session.NewRelayerSessions(
			ctx, deps,
			session.WithStoresDirectory(smtStorePath),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(relayerSessionsManager)), nil
	}
}

// uniqueSigningKeyNames goes through RelayMiner configuration and returns a list of unique
// operators singning key names.
func uniqueSigningKeyNames(relayMinerConfig *relayerconfig.RelayMinerConfig) []string {
	uniqueKeyMap := make(map[string]bool)
	for _, server := range relayMinerConfig.Servers {
		for _, supplier := range server.SupplierConfigsMap {
			for _, signingKeyName := range supplier.SigningKeyNames {
				uniqueKeyMap[signingKeyName] = true
			}
		}
	}

	uniqueKeyNames := make([]string, 0, len(uniqueKeyMap))
	for key := range uniqueKeyMap {
		uniqueKeyNames = append(uniqueKeyNames, key)
	}

	return uniqueKeyNames
}
