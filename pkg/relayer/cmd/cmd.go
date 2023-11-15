package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/signals"
	"github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
)

var (
	flagSigningKeyName string
	flagSmtStorePath   string
	flagSequencerNode  string
	flagPocketNode     string
)

type supplierFn func(
	context.Context,
	depinject.Config,
	*cobra.Command,
) (depinject.Config, error)

func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayminer",
		Short: "Run a relay miner",
		// TODO_TECHDEBT: add a longer long description.
		Long: `Run a relay miner`,
		RunE: runRelayer,
	}

	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")

	// TODO_TECHDEBT: integrate these flags with the client context (i.e. cosmosflags, config, viper, etc.)
	// This is simpler to do with server-side configs (see rootCmd#PersistentPreRunE) and requires more effort than currently justifiable.
	cmd.Flags().StringVar(&flagSigningKeyName, "signing-key", "", "Name of the key to sign transactions")
	// TODO_TECHDEBT(#137): This, alongside other flags, should be part of a config file suppliers provide.
	cmd.Flags().StringVar(&flagSmtStorePath, "smt-store", "smt", "Path to where the data backing SMT KV store exists on disk")
	// Communication flags
	// TODO_TECHDEBT: We're using `explicitly omitting default` so the relayer crashes if these aren't specified.
	// Figure out what good defaults should be post alpha.
	cmd.Flags().StringVar(&flagSequencerNode, "sequencer-node", "explicitly omitting default", "<host>:<port> to sequencer node to submit txs")
	cmd.Flags().StringVar(&flagPocketNode, "pocket-node", "explicitly omitting default", "<host>:<port> to full pocket node for reading data and listening for on-chain events")
	cmd.Flags().String(cosmosflags.FlagNode, "explicitly omitting default", "registering the default cosmos node flag; needed to initialize the cosmostx and query contexts correctly")

	// Set --node flag to the --sequencer-node for the client context
	err := cmd.Flags().Set(cosmosflags.FlagNode, fmt.Sprintf("tcp://%s", flagSequencerNode))
	if err != nil {
		panic(err)
	}

	return cmd
}

func runRelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	// Ensure context cancellation.
	defer cancelCtx()

	// Handle interrupt and kill signals asynchronously.
	signals.GoOnExitSignal(cancelCtx)

	// Sets up the following dependencies:
	// Miner, EventsQueryClient, BlockClient, cosmosclient.Context, TxFactory,
	// TxContext, TxClient, SupplierClient, RelayerProxy, RelayerSessionsManager.
	deps, err := setupRelayerDependencies(ctx, cmd)
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
) (deps depinject.Config, err error) {
	pocketNodeWebsocketUrl, err := getPocketNodeWebsocketUrl(cmd)
	if err != nil {
		return nil, err
	}

	supplierFuncs := []supplierFn{
		newSupplyEventsQueryClientFn(pocketNodeWebsocketUrl), // leaf
		newSupplyBlockClientFn(pocketNodeWebsocketUrl),
		supplyMiner,         // leaf
		supplyClientContext, // leaf
		supplyTxFactory,
		supplyTxContext,
		supplyTxClient,
		supplySupplierClient,
		supplyRelayerProxy,
		supplyRelayerSessionsManager,
	}

	// Initialize deps to with empty depinject config.
	deps = depinject.Configs()
	for _, supplyFn := range supplierFuncs {
		deps, err = supplyFn(ctx, deps, cmd)
	}

	return deps, nil
}

// getPocketNodeWebsocketUrl returns the websocket URL of the Pocket Node to
// connect to for subscribing to on-chain events.
func getPocketNodeWebsocketUrl(cmd *cobra.Command) (string, error) {
	pocketNodeURLStr, err := cmd.Flags().GetString(cosmosflags.FlagNode)
	if err != nil {
		return "", err
	}

	pocketNodeURL, err := url.Parse(pocketNodeURLStr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ws://%s/websocket", pocketNodeURL.Host), nil
}

// newSupplyEventsQueryClientFn constructs an EventsQueryClient instance and returns
// a new depinject.Config which is supplied with the given deps and the new
// EventsQueryClient.
func newSupplyEventsQueryClientFn(
	pocketNodeWebsocketUrl string,
) supplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketUrl)

		return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
	}
}

// newSupplyBlockClientFn returns a function with constructs a BlockClient instance
// with the given nodeURL and returns a new
// depinject.Config which is supplied with the given deps and the new
// BlockClient.
func newSupplyBlockClientFn(pocketNodeWebsocketUrl string) supplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		blockClient, err := block.NewBlockClient(ctx, deps, pocketNodeWebsocketUrl)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(blockClient)), nil
	}
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

// supplyClientContext constructs a cosmosclient.Context instance and returns a
// new depinject.Config which is supplied with the given deps and the new
// cosmosclient.Context.
func supplyClientContext(
	_ context.Context,
	deps depinject.Config,
	cmd *cobra.Command,
) (depinject.Config, error) {
	// NB: The implementation of #GetClientTxContext() is identical to that of
	// #GetClientQueryContext(); either is sufficient for usage in both querying
	// and transaction building and broadcasting.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(clientCtx))
	return deps, nil
}

// supplyTxFactory constructs a cosmostx.Factory instance and returns a new
// depinject.Config which is supplied with the given deps and the new
// cosmostx.Factory.
func supplyTxFactory(
	_ context.Context,
	deps depinject.Config,
	cmd *cobra.Command,
) (depinject.Config, error) {
	var clientCtx cosmosclient.Context
	if err := depinject.Inject(deps, &clientCtx); err != nil {
		return nil, err
	}

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

// supplyTxClient constructs a TxClient instance and returns a new
// depinject.Config which is supplied with the given deps and the new TxClient.
func supplyTxClient(
	ctx context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	txClient, err := tx.NewTxClient(
		ctx,
		deps,
		tx.WithSigningKeyName(flagSigningKeyName),
		// TODO_TECHDEBT: populate this from some config.
		tx.WithCommitTimeoutBlocks(tx.DefaultCommitTimeoutHeightOffset),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(txClient)), nil
}

// supplySupplierClient constructs a SupplierClient instance and returns a new
// depinject.Config which is supplied with the given deps and the new
// SupplierClient.
func supplySupplierClient(
	_ context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	supplierClient, err := supplier.NewSupplierClient(
		deps,
		supplier.WithSigningKeyName(flagSigningKeyName),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(supplierClient)), nil
}

// supplyRelayerProxy constructs a RelayerProxy instance and returns a new
// depinject.Config which is supplied with the given deps and the new
// RelayerProxy.
func supplyRelayerProxy(
	_ context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	// TODO_TECHDEBT: this should be populated from some relayerProxy config.
	// TODO_TECHDEBT(#179): this hostname should be updated to match that of the
	// in-tilt anvil service.
	anvilURL, err := url.Parse("http://localhost:8547/")
	if err != nil {
		return nil, err
	}

	proxiedServiceEndpoints := map[string]url.URL{
		"anvil": *anvilURL,
	}

	relayerProxy, err := proxy.NewRelayerProxy(
		deps,
		proxy.WithSigningKeyName(flagSigningKeyName),
		proxy.WithProxiedServicesEndpoints(proxiedServiceEndpoints),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(relayerProxy)), nil
}

// supplyRelayerSessionsManager constructs a RelayerSessionsManager instance
// and returns a new depinject.Config which is supplied with the given deps and
// the new RelayerSessionsManager.
func supplyRelayerSessionsManager(
	ctx context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	relayerSessionsManager, err := session.NewRelayerSessions(
		ctx, deps,
		session.WithStoresDirectory(flagSmtStorePath),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(relayerSessionsManager)), nil
}
