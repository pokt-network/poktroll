package cmd

import (
	"context"
	"log"
	"net/url"
	"os"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/deps/config"
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
	flagRelayMinerConfig string
)

func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayminer",
		Short: "Run a relay miner",
		Long: `Run a relay miner. The relay miner process configures and starts
relay servers for each service the supplier actor identified by --signing-key is
staked for (configured on-chain).

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
	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")
	cmd.Flags().
		String(cosmosflags.FlagNode, omittedDefaultFlagValue, "This flag is present to register the cosmos node flag, which is needed to initialise the comsos transaction and query context correctly. Ultimately the NetworkNodeUrl and QueryNodeUrl fields in the config file are used to build these contexts, and this flag's value isn't used.")

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

	relayMinerConfig, err := relayerconfig.ParseRelayMinerConfigs(configContent)
	if err != nil {
		return err
	}

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

	// Start the relay miner
	log.Println("INFO: Starting relay miner...")
	if err := relayMiner.Start(ctx); err != nil {
		return err
	}

	log.Println("INFO: Relay miner stopped; exiting")
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
	pocketNodeWebsocketUrl := relayMinerConfig.PocketNodeWebsocketUrl
	queryNodeUrl := relayMinerConfig.QueryNodeUrl.String()
	networkNodeUrl := relayMinerConfig.NetworkNodeUrl.String()
	signingKeyName := relayMinerConfig.SigningKeyName
	proxiedServiceEndpoints := relayMinerConfig.ProxiedServiceEndpoints
	smtStorePath := relayMinerConfig.SmtStorePath

	supplierFuncs := []config.SupplierFn{
		config.NewSupplyEventsQueryClientFn(pocketNodeWebsocketUrl), // leaf
		config.NewSupplyBlockClientFn(pocketNodeWebsocketUrl),
		supplyMiner, // leaf
		config.NewSupplyQueryClientContextFn(queryNodeUrl), // leaf
		config.NewSupplyTxClientContextFn(networkNodeUrl),  // leaf
		config.NewAccountQuerierFn(),
		config.NewApplicationQuerierFn(),
		config.NewSupplyRingCacheFn(),
		supplyTxFactory,
		supplyTxContext,
		newSupplyTxClientFn(signingKeyName),
		newSupplySupplierClientFn(signingKeyName),
		newSupplyRelayerProxyFn(signingKeyName, proxiedServiceEndpoints),
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
	mnr, err := miner.NewMiner()
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

// newSupplyTxClientFn returns a function which constructs a TxClient
// instance and returns a new depinject.Config which is supplied with
// the given deps and the new TxClient.
func newSupplyTxClientFn(signingKeyName string) config.SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		txClient, err := tx.NewTxClient(
			ctx,
			deps,
			tx.WithSigningKeyName(signingKeyName),
			// TODO_TECHDEBT: populate this from some config.
			tx.WithCommitTimeoutBlocks(tx.DefaultCommitTimeoutHeightOffset),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(txClient)), nil
	}
}

// newSupplySupplierClientFn returns a function which constructs a
// SupplierClient instance and returns a new depinject.Config which is
// supplied with the given deps and the new SupplierClient.
func newSupplySupplierClientFn(signingKeyName string) config.SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		supplierClient, err := supplier.NewSupplierClient(
			deps,
			supplier.WithSigningKeyName(signingKeyName),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(supplierClient)), nil
	}
}

// newSupplyRelayerProxyFn returns a function which constructs a
// RelayerProxy instance and returns a new depinject.Config which
// is supplied with the given deps and the new RelayerProxy.
func newSupplyRelayerProxyFn(
	signingKeyName string,
	proxiedServiceEndpoints map[string]*url.URL,
) config.SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayerProxy, err := proxy.NewRelayerProxy(
			deps,
			proxy.WithSigningKeyName(signingKeyName),
			proxy.WithProxiedServicesEndpoints(proxiedServiceEndpoints),
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
