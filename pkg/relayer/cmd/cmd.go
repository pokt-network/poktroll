package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

var (
	flagSigningKeyName string
	flagSmtStorePath   string
	flagSequencerNode  string
	flagPocketNode     string
)

func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayerminer",
		Short: "Run a relay miner",
		Long:  `Run a relay miner`,
		RunE:  runRelayer,
	}

	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")

	// TODO_TECHDEBT: integrate these cosmosflags with the client context (i.e. cosmosflags, config, viper, etc.)
	// This is simpler to do with server-side configs (see rootCmd#PersistentPreRunE).
	// Will require more effort than currently justifiable.
	cmd.Flags().StringVar(&flagSigningKeyName, "signing-key", "", "Name of the key to sign transactions")
	cmd.Flags().StringVar(&flagSmtStorePath, "smt-store", "smt", "Path to the SMT KV store")
	// Communication cosmosflags
	// TODO_TECHDEBT: We're using `explicitly omitting default` so the relayer crashes if these aren't specified. Figure out
	// Figure out what good defaults should be post alpha.
	cmd.Flags().StringVar(&flagSequencerNode, "sequencer-node", "explicitly omitting default", "<host>:<port> to sequencer node to submit txs")
	cmd.Flags().StringVar(&flagPocketNode, "pocket-node", "explicitly omitting default", "<host>:<port> to full pocket node for reading data and listening for on-chain events")
	cmd.Flags().String(cosmosflags.FlagNode, "explicitly omitting default", "registering the default cosmos node flag; needed to initialize the cosmostx and query contexts correctly")

	// Set --node flag to the --sequencer-node for the client context
	err := cmd.Flags().Set(cosmosflags.FlagNode, fmt.Sprintf("tcp://%s", flagSequencerNode))
	if err != nil {
		//return nil, err
		panic(err)
	}

	return cmd
}

func runRelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	// Ensure context cancellation.
	defer cancelCtx()

	// Sets up the following dependencies:
	// Miner, EventsQueryClient, BlockClient, TxClient, SupplierClient, RelayerProxy, RelayMiner dependencies.
	deps, err := setupRelayerDependencies(ctx, cmd)
	if err != nil {
		return err
	}

	var relayMiner relayer.RelayMiner
	if err := depinject.Inject(
		deps,
		&relayMiner,
	); err != nil {
		return err
	}

	// Handle interrupts in a goroutine.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		// Block until we receive an interrupt or kill signal (OS-agnostic)
		<-sigCh

		// Signal goroutines to stop
		cancelCtx()
	}()

	// Start the relay miner
	log.Println("INFO: Starting relay miner...")
	relayMiner.Start(ctx)

	log.Println("INFO: Relay miner stopped; exiting")
	return nil
}

// setupRelayerDependencies sets up all the dependencies the relay miner needs to run:
// Miner, EventsQueryClient, BlockClient, TxClient, SupplierClient, RelayerProxy, RelayMiner
func setupRelayerDependencies(
	ctx context.Context,
	cmd *cobra.Command,
) (deps depinject.Config, err error) {
	// Initizlize deps to with empty depinject config.
	deps, err = supplyMiner(depinject.Configs())
	if err != nil {
		return nil, err
	}

	rpcQueryURL, err := getPocketNodeWebsocketURL(cmd)
	if err != nil {
		return nil, err
	}

	// Has no dependencies.
	deps, err = supplyEventsQueryClient(deps, rpcQueryURL)
	if err != nil {
		return nil, err
	}

	// Depends on EventsQueryClient.
	deps, err = supplyBlockClient(ctx, deps, rpcQueryURL)
	if err != nil {
		return nil, err
	}

	// Has no dependencies.
	deps, err = supplyTxClientCtxAndTxFactory(deps, cmd)
	if err != nil {
		return nil, err
	}

	//var clientCtx cosmosclient.Context
	//if err := depinject.Inject(deps, &clientCtx); err != nil {
	//	panic(err)
	//}

	clientCtx, err := cosmosclient.GetClientQueryContext(cmd)
	if err != nil {
		panic(err)
	}
	supplierQuerier := suppliertypes.NewQueryClient(clientCtx)
	supplierQuery := &suppliertypes.QueryGetSupplierRequest{Address: ""}

	log.Printf("clientCtx: %+v", clientCtx)
	_, err = supplierQuerier.Supplier(ctx, supplierQuery)
	if err != nil {
		panic(err)
	}

	// Depends on clientCtx, txFactory, EventsQueryClient, & BlockClient.
	deps, err = supplyTxClient(ctx, deps)
	if err != nil {
		return nil, err
	}

	// Depends on txClient & EventsQueryClient.
	deps, err = supplySupplierClient(deps)
	if err != nil {
		return nil, err
	}

	// Depends on clientCtx & BlockClient.
	deps, err = supplyRelayerProxy(deps, cmd)
	if err != nil {
		return nil, err
	}

	// Depends on BlockClient & SupplierClient.
	deps, err = supplyRelayerSessionsManager(ctx, deps)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

func supplyMiner(
	deps depinject.Config,
) (depinject.Config, error) {
	mnr, err := miner.NewMiner()
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(mnr)), nil
}

func supplyEventsQueryClient(deps depinject.Config, pocketNodeWebsocketURL string) (depinject.Config, error) {
	eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketURL)

	return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
}

// TODO_IN_THIS_COMMIT: move
func getPocketNodeWebsocketURL(cmd *cobra.Command) (string, error) {
	pocketNodeURI, err := cmd.Flags().GetString(cosmosflags.FlagNode)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ws://%s/websocket", pocketNodeURI), nil
}

func supplyBlockClient(
	ctx context.Context,
	deps depinject.Config,
	nodeURL string,
) (depinject.Config, error) {
	blockClient, err := block.NewBlockClient(ctx, deps, nodeURL)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(blockClient)), nil
}

func supplyTxClientCtxAndTxFactory(
	deps depinject.Config,
	cmd *cobra.Command,
) (depinject.Config, error) {
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}
	clientFactory, err := cosmostx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		return nil, err
	}

	txClientCtx := relayer.TxClientContext(clientCtx)
	return depinject.Configs(deps, depinject.Supply(txClientCtx, clientFactory)), nil
}

func supplyTxClient(
	ctx context.Context,
	deps depinject.Config,
) (depinject.Config, error) {
	txContext, err := tx.NewTxContext(deps)
	if err != nil {
		return nil, err
	}

	deps = depinject.Configs(deps, depinject.Supply(txContext))
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

func supplySupplierClient(deps depinject.Config) (depinject.Config, error) {
	supplierClient, err := supplier.NewSupplierClient(
		deps,
		supplier.WithSigningKeyName(flagSigningKeyName),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(supplierClient)), nil
}

func supplyRelayerProxy(
	deps depinject.Config,
	cmd *cobra.Command,
) (depinject.Config, error) {
	// TODO_TECHDEBT: this should be populated from some relayerProxy config.
	anvilURL, err := url.Parse("ws://anvil:8547/")
	if err != nil {
		return nil, err
	}

	proxiedServiceEndpoints := map[string]url.URL{
		"anvil": *anvilURL,
	}

	clientCtx, err := cosmosclient.GetClientQueryContext(cmd)
	if err != nil {
		return nil, err
	}

	queryClientCtx := relayer.QueryClientContext(clientCtx)
	deps = depinject.Configs(deps, depinject.Supply(queryClientCtx))

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

func supplyRelayerSessionsManager(
	ctx context.Context,
	deps depinject.Config,
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
