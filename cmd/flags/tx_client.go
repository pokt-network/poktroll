package flags

import (
	"context"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/deps/config"
)

// GetTxClient constructs a new TxClient instance using the provided command flags.
func GetTxClient(
	ctx context.Context,
	cmd *cobra.Command,
	txClientOpts ...client.TxClientOption,
) (client.TxClient, error) {
	// Retrieve the query node RPC URL
	queryNodeRPCUrlString, err := cmd.Flags().GetString(cosmosflags.FlagNode)
	if err != nil {
		return nil, err
	}

	// Parse the query node RPC URL
	queryNodeRPCUrl, err := url.Parse(queryNodeRPCUrlString)
	if err != nil {
		return nil, err
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}

	// Conventionally construct a txClient and its dependencies
	clientFactory, err := cosmostx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		return nil, err
	}

	// Construct dependencies for the tx client
	deps, err := config.SupplyConfig(ctx, cmd, []config.SupplierFn{
		config.NewSupplyEventsQueryClientFn(queryNodeRPCUrl),
		config.NewSupplyBlockQueryClientFn(queryNodeRPCUrl),
		config.NewSupplyBlockClientFn(queryNodeRPCUrl),
	})
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(
		types.Context(clientCtx),
		clientFactory,
	))

	// Construct a tx client and inject its dependencies
	txCtx, err := tx.NewTxContext(deps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(txCtx))

	// Prepare default tx client options.
	gasAndFeesOptions, err := config.GetTxClientGasAndFeesOptions(cmd)
	if err != nil {
		return nil, err
	}

	defaultTxClientOpts := append(
		gasAndFeesOptions,
		tx.WithSigningKeyName(clientCtx.FromName),
	)

	// Prepend the default options such that are provided but can
	// be overridden with a subsequent options of the same kind.
	txClientOpts = append(defaultTxClientOpts, txClientOpts...)

	// Return a new TxClient instance
	return tx.NewTxClient(ctx, deps, txClientOpts...)
}
