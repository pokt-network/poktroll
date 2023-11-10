package cmd

import (
	"context"
	"fmt"
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
	// TODO_IN_THIS_COMMIT: ensure context is always cancelled.
	ctx, cancelCtx := context.WithCancel(cmd.Context())

	// Set --node flag to the --sequencer-node for the client context
	cmd.Flags().Set(cosmosflags.FlagNode, fmt.Sprintf("tcp://%s", sequencerNode))

	nodeURL, err := cmd.Flags().GetString(cosmosflags.FlagNode)
	if err != nil {
		return err
	}

	// Construct base dependency injection config.
	deps := supplyEventsQueryClient(nodeURL)
	deps, err = supplyBlockClient(ctx, deps, nodeURL)
	if err != nil {
		return err
	}

	deps, err = supplyTxClient(ctx, deps, cmd)
	if err != nil {
		return err
	}

	deps, err = supplySupplierClient(deps)
	if err != nil {
		return err
	}

	// -- BEGIN relayer proxy
	// INCOMPLETE: this should be populated from some relayer config.
	serviceEndpoints := map[string][]string{
		"svc1": {"ws://anvil:8547/"},
		"svc2": {"http://anvil:8547"},
	}

	relayer, err := proxy.NewRelayerProxy(
		deps,
		relayerProxyOpts,
	)
	if err != nil {
		cancelCtx()
		return err
	}
	//WithKey(ctx, clientFactory.Keybase(), signingKeyName, address.String(), clientCtx, servicerClient, serviceEndpoints).
	//WithServicerClient(servicerClient).
	//WithKVStorePath(ctx, filepath.Join(clientCtx.HomeDir, smtStorePath))

	if err := relayer.Start(ctx); err != nil {
		cancelCtx()
		return err
	}
	// -- END relayer proxy

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	// Block until we receive an interrupt or kill signal (OS-agnostic)
	<-sigCh

	// Signal goroutines to stop
	cancelCtx()
	// Wait for all goroutines to finish
	//wg.Wait()

	// TODO_IN_THIS_COMMIT: synchronize exit

	return nil
}

func supplyEventsQueryClient(nodeURL string) depinject.Config {
	eventsQueryClient := eventsquery.NewEventsQueryClient(nodeURL)
	return depinject.Supply(eventsQueryClient)
}

func supplyBlockClient(
	ctx context.Context,
	deps depinject.Config,
	nodeURL string) (depinject.Config, error) {
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

	deps = depinject.Supply(clientCtx, clientFactory)
	txContext, err := tx.NewTxContext(deps)
	if err != nil {
		return nil, err
	}

	deps = depinject.Configs(depinject.Supply(txContext))
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

	return depinject.Configs(depinject.Supply(txClient)), nil
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
	relayerProxy, err := proxy.NewRelayerProxy(deps)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(relayerProxy)), nil
}
