package config

import (
	"context"
	"fmt"
	"math"
	"net/url"

	"cosmossdk.io/depinject"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/relayer/relay_authenticator"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
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

// NewSupplyCometClientFn supplies a depinject config with an
// comet HTTP client from the given queryNodeRPCURL.
func NewSupplyCometClientFn(queryNodeRPCURL *url.URL) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {

		// Inject the logger from the deps
		var logger polylog.Logger
		err := depinject.Inject(deps, &logger)
		if err != nil {
			return nil, err
		}

		// Convert the query node RPC URL to a comet client
		cometClient, err := sdkclient.NewClientFromNode(queryNodeRPCURL.String())
		if err != nil {
			return nil, err
		}

		// Convert polylog logger to comet logger implementation:
		// - CometBFT client requires a logger implementing the CometBFT log.Logger interface
		// - Our application standardizes on polylog logger throughout the codebase
		// - The wrapper in polylog/comet_logger.go adapts between these interfaces
		// - This approach maintains consistent logging patterns across the application
		cometLogger := polylog.ToCometLogger(logger.With("component", "comet-client"))
		cometClient.SetLogger(cometLogger)

		// IMPORTANT: The CometBFT client MUST be started immediately after creation.
		// This ensures the client is fully initialized before any dependent components
		// attempt to use it for subscriptions, preventing connection errors.
		if err := cometClient.Start(); err != nil {
			return nil, err
		}

		// Inject the comet client into the deps
		return depinject.Configs(deps, depinject.Supply(cometClient)), nil
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
	return func(
		ctx context.Context,
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

		// Get the chain ID from the configured query client context.
		nodeStatus, err := cmtservice.GetNodeStatus(ctx, queryClientCtx)
		if err != nil {
			return nil, err
		}

		// Check if the network's returned chain ID matches the configured chain ID.
		if nodeStatus.NodeInfo.Network != queryClientCtx.ChainID {
			return nil, fmt.Errorf(
				"chain ID mismatch: client is configured for %q but the RPC node reports %q - ensure you're connecting to the correct network",
				queryClientCtx.ChainID,
				nodeStatus.NodeInfo.Network,
			)
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
//   - signingKeyNames is a list of operators signing key name corresponding to
//     the staked suppliers operator addresses.
//   - gasSettingStr is the gas setting to use for the tx client.
//     Options are "auto", "<integer>", or "".
//     See: config.GetTxClientGasAndFeesOptionsFromFlags.
func NewSupplySupplierClientsFn(signingKeyNames []string, gasSettingStr string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Set up the tx client options for the suppliers.
		txClientOptions, err := GetTxClientGasAndFeesOptionsFromFlags(cmd, gasSettingStr)
		if err != nil {
			return nil, err
		}

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
		queryCachingEnabled, err := cmd.Flags().GetBool(flags.FlagQueryCaching)
		if err != nil {
			return nil, err
		}

		// Use a NoOpKeyValueCache if query caching is disabled
		if !queryCachingEnabled {
			noopParamsCache := querycache.NewNoOpKeyValueCache[T]()
			return depinject.Configs(deps, depinject.Supply(noopParamsCache)), nil
		}

		kvCache, err := memory.NewKeyValueCache[T](memory.WithTTL(math.MaxInt64))
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
		queryCachingEnabled, err := cmd.Flags().GetBool(flags.FlagQueryCaching)
		if err != nil {
			return nil, err
		}

		// Use a NoOpParamsCache if query caching is disabled
		if !queryCachingEnabled {
			noopParamsCache := querycache.NewNoOpParamsCache[T]()
			return depinject.Configs(deps, depinject.Supply(noopParamsCache)), nil
		}

		// TODO_TECHDEBT(red-0ne) Set ttl to block time + some buffer time when we
		// switch to event-driven cache warming.
		paramsCache, err := querycache.NewParamsCache[T](memory.WithTTL(math.MaxInt64))
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

// SupplyMiner constructs a Miner instance and returns a new depinject.Config with it supplied.
//
// - Supplies Miner to the dependency injection config
// - Returns updated config and error if any
//
// Parameters:
//   - ctx: Context for the function
//   - deps: Dependency injection config
//   - cmd: Cobra command
//
// Returns:
//   - depinject.Config: Updated dependency injection config
//   - error: Error if setup fails
func SupplyMiner(
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

// SupplyRelayMeterFn returns a function which constructs a RelayMeter instance
// and returns a new depinject.Config with it supplied.
//
// - Accepts enableOverServicing boolean for proxy setup
// - Returns a SupplierFn for dependency injection
//
// Parameters:
//   - enableOverServicing: Enable over-servicing in the relay meter
//
// Returns:
//   - SupplierFn: Supplier function for dependency injection
func SupplyRelayMeterFn(
	enableOverServicing bool,
) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		rm, err := proxy.NewRelayMeter(deps, enableOverServicing)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(rm)), nil
	}
}

// SupplyTxFactory constructs a cosmostx.Factory instance and returns a new depinject.Config with it supplied.
//
// - Supplies TxFactory to the dependency injection config
// - Returns updated config and error if any
//
// Parameters:
//   - ctx: Context for the function
//   - deps: Dependency injection config
//   - cmd: Cobra command
//
// Returns:
//   - depinject.Config: Updated dependency injection config
//   - error: Error if setup fails
func SupplyTxFactory(
	_ context.Context,
	deps depinject.Config,
	cmd *cobra.Command,
) (depinject.Config, error) {
	var txClientCtx txtypes.Context
	if err := depinject.Inject(deps, &txClientCtx); err != nil {
		return nil, err
	}

	clientCtx := sdkclient.Context(txClientCtx)
	clientFactory, err := cosmostx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(clientFactory)), nil
}

// SupplyTxContext constructs a transaction context and returns a new depinject.Config with it supplied.
//
// - Supplies TxContext to the dependency injection config
// - Returns updated config and error if any
//
// Parameters:
//   - ctx: Context for the function
//   - deps: Dependency injection config
//   - cmd: Cobra command
//
// Returns:
//   - depinject.Config: Updated dependency injection config
//   - error: Error if setup fails
func SupplyTxContext(
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

// NewSupplyRelayAuthenticatorFn returns a function which constructs a RelayAuthenticator and returns a new depinject.Config with it supplied.
//
// - Accepts signingKeyNames for authenticator setup
// - Returns a SupplierFn for dependency injection
//
// Parameters:
//   - signingKeyNames: List of signing key names
//
// Returns:
//   - SupplierFn: Supplier function for dependency injection
func NewSupplyRelayAuthenticatorFn(
	signingKeyNames []string,
) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayAuthenticator, err := relay_authenticator.NewRelayAuthenticator(
			deps,
			relay_authenticator.WithSigningKeyNames(signingKeyNames),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(relayAuthenticator)), nil
	}
}

// newSupplyRelayerProxyFn returns a function which constructs a RelayerProxy and returns a new depinject.Config with it supplied.
//
//   - Accepts servicesConfigMap for proxy setup
//   - Accepts pingEnabled flag to enable pinging the backend services to ensure
//     they are correctly setup and reachable before starting the relayer proxy.
//   - Returns a SupplierFn for dependency injection
//
// Parameters:
//   - servicesConfigMap: Map of services configuration
//   - pingEnabled: Flag to enable pinging the backend services
//
// Returns:
//   - SupplierFn: Supplier function for dependency injection
func NewSupplyRelayerProxyFn(
	servicesConfigMap map[string]*relayerconfig.RelayMinerServerConfig,
	pingEnabled bool,
) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayerProxy, err := proxy.NewRelayerProxy(
			deps,
			proxy.WithServicesConfigMap(servicesConfigMap),
			proxy.WithPingEnabled(pingEnabled),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(relayerProxy)), nil
	}
}

// newSupplyRelayerSessionsManagerFn returns a function which constructs a RelayerSessionsManager and returns a new depinject.Config with it supplied.
//
// - Accepts smtStorePath for sessions manager setup
// - Returns a SupplierFn for dependency injection
//
// Parameters:
//   - smtStorePath: Path to the sessions store
//
// Returns:
//   - config.SupplierFn: Supplier function for dependency injection
func NewSupplyRelayerSessionsManagerFn(smtStorePath string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayerSessionsManager, err := session.NewRelayerSessions(
			deps,
			session.WithStoresDirectory(smtStorePath),
		)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(relayerSessionsManager)), nil
	}
}
