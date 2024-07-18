package config

import (
	"context"
	"net/url"

	"cosmossdk.io/depinject"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/appgateserver/sdkadapter"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/proof"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	"github.com/pokt-network/poktroll/pkg/client/session"
	"github.com/pokt-network/poktroll/pkg/client/shared"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// SupplierFn is a function that is used to supply a depinject config.
type SupplierFn func(
	context.Context,
	depinject.Config,
	*cobra.Command,
) (depinject.Config, error)

// SupplyConfig supplies a depinject config by calling each of the supplied
// supplier functions in order and passing the result of each supplier to the
// next supplier, chaining them together.
func SupplyConfig(
	ctx context.Context,
	cmd *cobra.Command,
	suppliers []SupplierFn,
) (deps depinject.Config, err error) {
	// Initialize deps to with empty depinject config.
	deps = depinject.Configs()
	for _, supplyFn := range suppliers {
		deps, err = supplyFn(ctx, deps, cmd)
		if err != nil {
			return nil, err
		}
	}
	return deps, nil
}

// NewSupplyLoggerFromCtx supplies a depinject config with a polylog.Logger instance
// populated from the given context.
func NewSupplyLoggerFromCtx(ctx context.Context) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		return depinject.Configs(deps, depinject.Supply(polylog.Ctx(ctx))), nil
	}
}

// NewSupplyEventsQueryClientFn supplies a depinject config with an
// EventsQueryClient from the given queryNodeRPCURL.
func NewSupplyEventsQueryClientFn(queryNodeRPCURL *url.URL) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Convert the host to a websocket URL
		queryNodeWebsocketURL := events.RPCToWebsocketURL(queryNodeRPCURL)
		eventsQueryClient := events.NewEventsQueryClient(queryNodeWebsocketURL)

		return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
	}
}

// NewSupplyBlockClientFn supplies a depinject config with a blockClient.
func NewSupplyBlockClientFn(queryNodeRPCURL *url.URL) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {

		// Requires a query client to be supplied to the deps
		blockClient, err := block.NewBlockClient(ctx, deps)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(blockClient)), nil
	}
}

// NewSupplyDelegationClientFn returns a function which constructs a
// DelegationClient instance and returns a new depinject.Config which is
// supplied with the given deps and the new DelegationClient.
func NewSupplyDelegationClientFn() SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Requires a query client to be supplied to the deps
		delegationClient, err := delegation.NewDelegationClient(ctx, deps)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(delegationClient)), nil
	}
}

// NewSupplyQueryClientContextFn supplies a depinject config with a query
//
//	ClientContext, a GRPC client connection, and a keyring from the given queryNodeGRPCURL.
func NewSupplyQueryClientContextFn(queryNodeGRPCURL *url.URL) SupplierFn {
	return func(_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Temporarily store the flag's current value to be restored later, after
		// the client context has been created with queryNodeGRPCURL.
		// TODO_TECHDEBT(#223) Retrieve value from viper instead, once integrated.
		tmpGRPC, err := cmd.Flags().GetString(cosmosflags.FlagGRPC)
		if err != nil {
			return nil, err
		}

		// Set --grpc-addr flag to the pocketQueryNodeURL for the client context
		// This flag is read by sdkclient.GetClientQueryContext.
		// Cosmos-SDK is expecting a GRPC address formatted as <hostname>[:<port>],
		// so we only need to set the Host parameter of the URL to cosmosflags.FlagGRPC value.
		if err = cmd.Flags().Set(cosmosflags.FlagGRPC, queryNodeGRPCURL.Host); err != nil {
			return nil, err
		}

		// NB: Currently, the implementations of GetClientTxContext() and
		// GetClientQueryContext() are identical, allowing for their interchangeable
		// use in both querying and transaction operations. However, in order to support
		// independent configuration of client contexts for distinct querying and
		// transacting purposes.
		// For example, txs could be dispatched to a validator while queries
		// could be handled by a full-node.
		queryClientCtx, err := sdkclient.GetClientQueryContext(cmd)
		if err != nil {
			return nil, err
		}
		deps = depinject.Configs(deps, depinject.Supply(
			querytypes.Context(queryClientCtx),
			grpc.ClientConn(queryClientCtx),
			queryClientCtx.Keyring,
		))

		// Restore the flag's original value in order for other components
		// to use the flag as expected.
		if err := cmd.Flags().Set(cosmosflags.FlagGRPC, tmpGRPC); err != nil {
			return nil, err
		}

		return deps, nil
	}
}

// NewSupplyTxClientContextFn supplies a depinject config with a TxClientContext
// from the given txNodeGRPCURL.
// TODO_TECHDEBT(#256): Remove this function once the as we may no longer
// need to supply a TxClientContext to the RelayMiner.
func NewSupplyTxClientContextFn(
	queryNodeGRPCURL *url.URL,
	txNodeRPCURL *url.URL,
) SupplierFn {
	return func(_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Temporarily store the flag's current value to be restored later, after
		// the client context has been created with txNodeRPCURL.
		// TODO_TECHDEBT(#223) Retrieve value from viper instead, once integrated.
		tmpNode, err := cmd.Flags().GetString(cosmosflags.FlagNode)
		if err != nil {
			return nil, err
		}

		// Temporarily store the flag's current value to be restored later, after
		// the client context has been created with queryNodeGRPCURL.
		// TODO_TECHDEBT(#223) Retrieve value from viper instead, once integrated.
		tmpGRPC, err := cmd.Flags().GetString(cosmosflags.FlagGRPC)
		if err != nil {
			return nil, err
		}

		// Set --node flag to the txNodeRPCURL for the client context
		// This flag is read by sdkclient.GetClientTxContext.
		if err = cmd.Flags().Set(cosmosflags.FlagNode, txNodeRPCURL.String()); err != nil {
			return nil, err
		}

		// Set --grpc-addr flag to the queryNodeGRPCURL for the client context
		// This flag is read by sdkclient.GetClientTxContext to query accounts
		// for transaction signing.
		// Cosmos-SDK is expecting a GRPC address formatted as <hostname>[:<port>],
		// so we only need to set the Host parameter of the URL to cosmosflags.FlagGRPC value.
		if err = cmd.Flags().Set(cosmosflags.FlagGRPC, queryNodeGRPCURL.Host); err != nil {
			return nil, err
		}

		tmpChainID, err := cmd.Flags().GetString(cosmosflags.FlagChainID)
		if err != nil {
			return nil, err
		}

		if err = cmd.Flags().Set(cosmosflags.FlagChainID, tmpChainID); err != nil {
			return nil, err
		}

		// NB: Currently, the implementations of GetClientTxContext() and
		// GetClientQueryContext() are identical, allowing for their interchangeable
		// use in both querying and transaction operations. However, in order to support
		// independent configuration of client contexts for distinct querying and
		// transacting purposes.
		// For example, txs could be dispatched to a validator while queries
		// could be handled by a full-node
		txClientCtx, err := sdkclient.GetClientTxContext(cmd)
		if err != nil {
			return nil, err
		}
		deps = depinject.Configs(deps, depinject.Supply(
			txtypes.Context(txClientCtx),
		))

		// Restore the flag's original value in order for other components
		// to use the flag as expected.
		if err := cmd.Flags().Set(cosmosflags.FlagGRPC, tmpGRPC); err != nil {
			return nil, err
		}

		// Restore the flag's original value in order for other components
		// to use the flag as expected.
		if err := cmd.Flags().Set(cosmosflags.FlagNode, tmpNode); err != nil {
			return nil, err
		}

		if err := cmd.Flags().Set(cosmosflags.FlagChainID, tmpChainID); err != nil {
			return nil, err
		}

		return deps, nil
	}
}

// NewSupplyAccountQuerierFn supplies a depinject config with an AccountQuerier.
func NewSupplyAccountQuerierFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the account querier.
		accountQuerier, err := query.NewAccountQuerier(deps)
		if err != nil {
			return nil, err
		}

		// Supply the account querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(accountQuerier)), nil
	}
}

// NewSupplyApplicationQuerierFn supplies a depinject config with an ApplicationQuerier.
func NewSupplyApplicationQuerierFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the application querier.
		applicationQuerier, err := query.NewApplicationQuerier(deps)
		if err != nil {
			return nil, err
		}

		// Supply the application querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(applicationQuerier)), nil
	}
}

// NewSupplySessionQuerierFn supplies a depinject config with a SessionQuerier.
func NewSupplySessionQuerierFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the session querier.
		sessionQuerier, err := session.NewSessionQueryClient(deps)
		if err != nil {
			return nil, err
		}

		// Supply the session querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(sessionQuerier)), nil
	}
}

// NewSupplySupplierQuerierFn supplies a depinject config with a SupplierQuerier.
func NewSupplySupplierQuerierFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the supplier querier.
		supplierQuerier, err := query.NewSupplierQuerier(deps)
		if err != nil {
			return nil, err
		}

		// Supply the supplier querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(supplierQuerier)), nil
	}
}

// NewSupplyRingCacheFn supplies a depinject config with a RingCache.
func NewSupplyRingCacheFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the ring cache.
		ringCache, err := rings.NewRingCache(deps)
		if err != nil {
			return nil, err
		}

		// Supply the ring cache to the provided deps
		return depinject.Configs(deps, depinject.Supply(ringCache)), nil
	}
}

// NewSupplyShannonSDKFn supplies a depinject config with a ShannonSDK given
// the signing key name.
func NewSupplyShannonSDKFn(signingKeyName string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		var clientCtx sdkclient.Context

		// On a Cosmos environment we get the private key from the keyring
		// Inject the client context, get the keyring from it then get the private key
		if err := depinject.Inject(deps, &clientCtx); err != nil {
			return nil, err
		}

		keyRecord, err := clientCtx.Keyring.Key(signingKeyName)
		if err != nil {
			return nil, err
		}

		privateKey, ok := keyRecord.GetLocal().PrivKey.GetCachedValue().(cryptotypes.PrivKey)
		if !ok {
			return nil, err
		}

		shannonSDK, err := sdkadapter.NewShannonSDK(ctx, privateKey, deps)
		if err != nil {
			return nil, err
		}

		// Supply the session querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(shannonSDK)), nil
	}
}

// NewSupplySupplierClientsFn returns a function which constructs a
// SupplierClientMap and returns a new depinject.Config which is
// supplied with the given deps and the new SupplierClientMap.
func NewSupplySupplierClientsFn(signingKeyNames []string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		suppliers := proof.NewSupplierClientMap()
		for _, signingKeyName := range signingKeyNames {
			txClientDepinjectConfig, err := newSupplyTxClientsFn(ctx, deps, signingKeyName)
			if err != nil {
				return nil, err
			}

			supplierClient, err := proof.NewProofClient(
				txClientDepinjectConfig,
				proof.WithSigningKeyName(signingKeyName),
			)
			if err != nil {
				return nil, err
			}

			// Making sure we use addresses as keys.
			suppliers.SupplierClients[supplierClient.Address().String()] = supplierClient
		}
		return depinject.Configs(deps, depinject.Supply(suppliers)), nil
	}
}

// NewSupplyBlockQueryClientFn returns a function which constructs a
// BlockQueryClient instance and returns a new depinject.Config which
// is supplied with the given deps and the new BlockQueryClient.
func NewSupplyBlockQueryClientFn(queryNodeRPCUrl *url.URL) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		blockQueryClient, err := sdkclient.NewClientFromNode(queryNodeRPCUrl.String())
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(blockQueryClient)), nil
	}
}

// NewSupplySharedQueryClientFn returns a function which constructs a
// SharedQueryClient instance and returns a new depinject.Config which
// is supplied with the given deps and the new SharedQueryClient.
func NewSupplySharedQueryClientFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		sharedQuerier, err := shared.NewSharedQueryClient(deps)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(sharedQuerier)), nil
	}
}

// NewSupplyProofQueryClientFn returns a function which constructs a
// ProofQueryClient instance and returns a new depinject.Config which
// is supplied with the given deps and the new ProofQueryClient.
func NewSupplyProofQueryClientFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		proofQuerier, err := proof.NewProofQueryClient(deps)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(proofQuerier)), nil
	}
}

// newSupplyTxClientFn returns a new depinject.Config which is supplied with
// the given deps and the new TxClient.
func newSupplyTxClientsFn(ctx context.Context, deps depinject.Config, signingKeyName string) (depinject.Config, error) {
	txClient, err := tx.NewTxClient(
		ctx,
		deps,
		tx.WithSigningKeyName(signingKeyName),
		tx.WithCommitTimeoutBlocks(tx.DefaultCommitTimeoutHeightOffset),
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(txClient)), nil
}
