package config

import (
	"context"
	"fmt"
	"net/url"

	"cosmossdk.io/depinject"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// FlagQueryCaching is the flag name to enable or disable query caching.
const FlagQueryCaching = "query-caching"

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
			query.Context(queryClientCtx),
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
		sessionQuerier, err := query.NewSessionQuerier(deps)
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

// NewSupplyRingClientFn supplies a depinject config with a RingClient.
func NewSupplyRingClientFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the ring client.
		ringClient, err := rings.NewRingClient(deps)
		if err != nil {
			return nil, err
		}

		// Supply the ring cache to the provided deps
		return depinject.Configs(deps, depinject.Supply(ringClient)), nil
	}
}

// NewSupplySupplierClientsFn returns a function which constructs a
// SupplierClientMap and returns a new depinject.Config which is
// supplied with the given deps and the new SupplierClientMap.
// - signingKeyNames is a list of operators signing key name corresponding to
// the staked suppliers operator addresess.
func NewSupplySupplierClientsFn(signingKeyNames []string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		gasPriceStr, err := cmd.Flags().GetString(cosmosflags.FlagGasPrices)
		if err != nil {
			return nil, err
		}

		gasPrices, err := cosmostypes.ParseDecCoins(gasPriceStr)
		if err != nil {
			return nil, err
		}

		gasAdjustment, err := cmd.Flags().GetFloat64(cosmosflags.FlagGasAdjustment)
		if err != nil {
			return nil, err
		}

		gasStr, err := cmd.Flags().GetString(cosmosflags.FlagGas)
		if err != nil {
			return nil, err
		}

		gasSetting, err := flags.ParseGasSetting(gasStr)
		if err != nil {
			return nil, err
		}

		feeAmountStr, err := cmd.Flags().GetString(cosmosflags.FlagFees)
		if err != nil {
			return nil, err
		}

		var feeAmount cosmostypes.DecCoins
		if feeAmountStr != "" {
			feeAmount, err = cosmostypes.ParseDecCoins(feeAmountStr)
			if err != nil {
				return nil, err
			}
		}

		// Ensure that the gas prices include upokt
		for _, gasPrice := range gasPrices {
			if gasPrice.Denom != volatile.DenomuPOKT {
				// TODO_TECHDEBT(red-0ne): Allow other gas prices denominations once supported (e.g. mPOKT, POKT)
				// See https://docs.cosmos.network/main/build/architecture/adr-024-coin-metadata#decision
				return nil, fmt.Errorf("only gas prices with %s denom are supported", volatile.DenomuPOKT)
			}
		}

		// Setup the tx client options for the suppliers.
		txClientOptions := make([]client.TxClientOption, 0)
		txClientOptions = append(txClientOptions, tx.WithGasPrices(&gasPrices))
		txClientOptions = append(txClientOptions, tx.WithGasAdjustment(gasAdjustment))
		txClientOptions = append(txClientOptions, tx.WithGasSetting(&gasSetting))
		txClientOptions = append(txClientOptions, tx.WithFeeAmount(&feeAmount))

		suppliers := supplier.NewSupplierClientMap()
		for _, signingKeyName := range signingKeyNames {
			txClientOptions = append(txClientOptions, tx.WithSigningKeyName(signingKeyName))
			txClientDepinjectConfig, err := newSupplyTxClientsFn(
				ctx,
				deps,
				txClientOptions...,
			)
			if err != nil {
				return nil, err
			}

			supplierClient, err := supplier.NewSupplierClient(
				txClientDepinjectConfig,
				supplier.WithSigningKeyName(signingKeyName),
			)
			if err != nil {
				return nil, err
			}

			// Making sure we use addresses as keys.
			suppliers.SupplierClients[supplierClient.OperatorAddress()] = supplierClient
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
		sharedQuerier, err := query.NewSharedQuerier(deps)
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
		proofQuerier, err := query.NewProofQuerier(deps)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(proofQuerier)), nil
	}
}

// NewSupplyServiceQueryClientFn returns a function which constructs a
// NewSupplyServiceQueryClient instance and returns a new depinject.Config which
// is supplied with the given deps and the new ServiceQueryClient.
func NewSupplyServiceQueryClientFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		serviceQuerier, err := query.NewServiceQuerier(deps)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(serviceQuerier)), nil
	}
}

// NewSupplyBankQuerierFn supplies a depinject config with an BankQuerier.
func NewSupplyBankQuerierFn() SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Create the bank querier.
		bankQuerier, err := query.NewBankQuerier(deps)
		if err != nil {
			return nil, err
		}

		// Supply the bank querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(bankQuerier)), nil
	}
}

// newSupplyTxClientFn returns a new depinject.Config which is supplied with
// the given deps and the new TxClient.
func newSupplyTxClientsFn(
	ctx context.Context,
	deps depinject.Config,
	txClientOptions ...client.TxClientOption,
) (depinject.Config, error) {

	txClient, err := tx.NewTxClient(
		ctx,
		deps,
		txClientOptions...,
	)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(txClient)), nil
}

// NewSupplyKeyValueCacheFn returns a function which constructs a KeyValueCache of type T.
// It take a list of cache options that can be used to configure the cache.
func NewSupplyKeyValueCacheFn[T any](opts ...querycache.CacheOption[cache.KeyValueCache[T]]) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Check if query caching is enabled
		queryCachingEnabled, err := cmd.Flags().GetBool(FlagQueryCaching)
		if err != nil {
			return nil, err
		}

		// Use a NoOpKeyValueCache if query caching is disabled
		if !queryCachingEnabled {
			noopParamsCache := querycache.NewNoOpKeyValueCache[T]()
			return depinject.Configs(deps, depinject.Supply(noopParamsCache)), nil
		}

		kvCache, err := memory.NewKeyValueCache[T](memory.WithTTL(0))
		if err != nil {
			return nil, err
		}

		// Apply the query cache options
		for _, opt := range opts {
			if err := opt(ctx, deps, kvCache); err != nil {
				return nil, err
			}
		}

		return depinject.Configs(deps, depinject.Supply(kvCache)), nil
	}
}

// NewSupplyParamsCacheFn returns a function which constructs a ParamsCache of type T.
// It take a list of cache options that can be used to configure the cache.
func NewSupplyParamsCacheFn[T any](opts ...querycache.CacheOption[client.ParamsCache[T]]) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Check if params caching is enabled
		queryCachingEnabled, err := cmd.Flags().GetBool(FlagQueryCaching)
		if err != nil {
			return nil, err
		}

		// Use a NoOpParamsCache if query caching is disabled
		if !queryCachingEnabled {
			noopParamsCache := querycache.NewNoOpParamsCache[T]()
			return depinject.Configs(deps, depinject.Supply(noopParamsCache)), nil
		}

		paramsCache, err := querycache.NewParamsCache[T](memory.WithTTL(0))
		if err != nil {
			return nil, err
		}

		// Apply the query cache options
		for _, opt := range opts {
			if err := opt(ctx, deps, paramsCache); err != nil {
				return nil, err
			}
		}

		return depinject.Configs(deps, depinject.Supply(paramsCache)), nil
	}
}
