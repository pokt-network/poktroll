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
)

var (
	signingKeyName string
	smtStorePath   string
	sequencerNode  string
	pocketNode     string
)

func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		// TODO_DISCUSS: do we want to rename this to `relay-miner`?
		Use:   "relayer",
		Short: "Run a relayer",
		Long:  `Run a relayer`,
		RunE:  runRelayer,
	}

	cmd.Flags().String(cosmosflags.FlagKeyringBackend, "", "Select keyring's backend (os|file|kwallet|pass|test)")

	// TECHDEBT: integrate these cosmosflags with the client context (i.e. cosmosflags, config, viper, etc.)
	// This is simpler to do with server-side configs (see rootCmd#PersistentPreRunE).
	// Will require more effort than currently justifiable.
	cmd.Flags().StringVar(&signingKeyName, "signing-key", "", "Name of the key to sign transactions")
	cmd.Flags().StringVar(&smtStorePath, "smt-store", "smt", "Path to the SMT KV store")
	// Communication cosmosflags
	// TODO_DISCUSS: We're using `explicitly omitting default` so the relayer crashes if these aren't specified. Figure out
	// what the defaults should be post alpha.
	cmd.Flags().StringVar(&sequencerNode, "sequencer-node", "explicitly omitting default", "<host>:<port> to sequencer/validator node to submit txs")
	cmd.Flags().StringVar(&pocketNode, "pocket-node", "explicitly omitting default", "<host>:<port> to full/light pocket node for reading data and listening for on-chain events")
	cmd.Flags().String(cosmosflags.FlagNode, "explicitly omitting default", "registering the default cosmos node flag; needed to initialize the cosmostx and query contexts correctly")

	return cmd
}

func runRelayer(cmd *cobra.Command, _ []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	// Ensure context cancellation.
	defer cancelCtx()

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

func setupRelayerDependencies(
	ctx context.Context,
	cmd *cobra.Command,
) (deps depinject.Config, err error) {
	// Set --node flag to the --sequencer-node for the client context
	cmd.Flags().Set(cosmosflags.FlagNode, fmt.Sprintf("tcp://%s", sequencerNode))

	nodeURL, err := cmd.Flags().GetString(cosmosflags.FlagNode)
	if err != nil {
		return nil, err
	}

	deps, err = supplyMiner(deps)
	if err != nil {
		return nil, err
	}

	deps = supplyEventsQueryClient(deps, nodeURL)

	deps, err = supplyBlockClient(ctx, deps, nodeURL)
	if err != nil {
		return nil, err
	}

	deps, err = supplyTxClient(ctx, deps, cmd)
	if err != nil {
		return nil, err
	}

	deps, err = supplySupplierClient(deps)
	if err != nil {
		return nil, err
	}

	deps, err = supplyRelayerProxy(deps)
	if err != nil {
		return nil, err
	}

	deps, err = supplyRelayMiner(ctx, deps)
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

func supplyEventsQueryClient(deps depinject.Config, nodeURL string) depinject.Config {
	eventsQueryClient := eventsquery.NewEventsQueryClient(nodeURL)

	return depinject.Configs(deps, depinject.Supply(eventsQueryClient))
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

func supplyTxClient(
	ctx context.Context,
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

	deps = depinject.Configs(deps, depinject.Supply(clientCtx, clientFactory))
	txContext, err := tx.NewTxContext(deps)
	if err != nil {
		return nil, err
	}

	deps = depinject.Configs(deps, depinject.Supply(txContext))
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

func supplySupplierClient(deps depinject.Config) (depinject.Config, error) {
	supplierClient, err := supplier.NewSupplierClient(
		deps,
		supplier.WithSigningKeyName(signingKeyName),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(supplierClient)), nil
}

func supplyRelayerProxy(deps depinject.Config) (depinject.Config, error) {
	// TODO_INCOMPLETE: this should be populated from some relayerProxy config.
	anvilURL, err := url.Parse("ws://anvil:8547/")
	if err != nil {
		return nil, err
	}

	proxiedServiceEndpoints := map[string]url.URL{
		"svc1": *anvilURL,
	}

	relayerProxy, err := proxy.NewRelayerProxy(
		deps,
		proxy.WithProxiedServicesEndpoints(proxiedServiceEndpoints),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(relayerProxy)), nil
}

func supplyRelayMiner(ctx context.Context, deps depinject.Config) (depinject.Config, error) {
	relayMiner, err := relayer.NewRelayMiner(ctx, deps)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(relayMiner)), nil
}
