package cmd

import (
	"context"
	"fmt"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/miner"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/relayer/relay_authenticator"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// setupRelayerDependencies builds and returns the dependency tree for the relay miner.
//
// - Builds from leaves up, incrementally supplying each component to depinject.Config
// - Sets up dependencies for various things that included but not limited to query clients, tx handlers, etc..
//
// Returns:
//   - deps: The dependency injection config
//   - err: Error if setup fails
func setupRelayerDependencies(
	ctx context.Context,
	cmd *cobra.Command,
	relayMinerConfig *relayerconfig.RelayMinerConfig,
) (deps depinject.Config, err error) {
	queryNodeRPCUrl := relayMinerConfig.PocketNode.QueryNodeRPCUrl
	queryNodeGRPCUrl := relayMinerConfig.PocketNode.QueryNodeGRPCUrl
	txNodeRPCUrl := relayMinerConfig.PocketNode.TxNodeRPCUrl

	// Override config file's `QueryNodeGRPCUrl` with `--grpc-addr` flag if specified.
	// TODO(#223): Remove this check once viper is used as SoT for overridable config values.
	if flagNodeGRPCURL != flags.OmittedDefaultFlagValue {
		parsedFlagNodeGRPCUrl, err := url.Parse(flagNodeGRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grpc query URL: %w", err)
		}
		queryNodeGRPCUrl = parsedFlagNodeGRPCUrl
	}

	// Override config file's `QueryNodeUrl` and `txNodeRPCUrl` with `--node` flag if specified.
	// TODO(#223): Remove this check once viper is used as SoT for overridable config values.
	if flagNodeRPCURL != flags.OmittedDefaultFlagValue {
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

		// Setup params caches (clear on new blocks).
		// Tokenomics/gateway params not used in RelayMiner, so no cache needed.
		config.NewSupplyParamsCacheFn[sharedtypes.Params](cache.WithNewBlockCacheClearing),   // leaf
		config.NewSupplyParamsCacheFn[apptypes.Params](cache.WithNewBlockCacheClearing),      // leaf
		config.NewSupplyParamsCacheFn[sessiontypes.Params](cache.WithNewBlockCacheClearing),  // leaf
		config.NewSupplyParamsCacheFn[prooftypes.Params](cache.WithNewBlockCacheClearing),    // leaf
		config.NewSupplyParamsCacheFn[servicetypes.Params](cache.WithNewBlockCacheClearing),  // leaf
		config.NewSupplyParamsCacheFn[suppliertypes.Params](cache.WithNewBlockCacheClearing), // leaf

		// Setup key-value caches for pocket types (clear on new blocks).
		config.NewSupplyKeyValueCacheFn[sharedtypes.Service](cache.WithNewBlockCacheClearing),                // leaf
		config.NewSupplyKeyValueCacheFn[servicetypes.RelayMiningDifficulty](cache.WithNewBlockCacheClearing), // leaf
		config.NewSupplyKeyValueCacheFn[apptypes.Application](cache.WithNewBlockCacheClearing),               // leaf
		config.NewSupplyKeyValueCacheFn[sharedtypes.Supplier](cache.WithNewBlockCacheClearing),               // leaf
		config.NewSupplyKeyValueCacheFn[query.BlockHash](cache.WithNewBlockCacheClearing),                    // leaf
		config.NewSupplyKeyValueCacheFn[query.Balance](cache.WithNewBlockCacheClearing),                      // leaf
		config.NewSupplyKeyValueCacheFn[prooftypes.Claim](cache.WithNewBlockCacheClearing),                   // leaf
		// Session querier returns *sessiontypes.Session, so cache must return pointers.
		config.NewSupplyKeyValueCacheFn[*sessiontypes.Session](cache.WithNewBlockCacheClearing), // leaf

		// Setup key-value for cosmos types (clear on new blocks).
		config.NewSupplyKeyValueCacheFn[cosmostypes.AccountI](cache.WithNewBlockCacheClearing), // leaf

		config.NewSupplySharedQueryClientFn(),
		config.NewSupplyServiceQueryClientFn(),
		config.NewSupplyApplicationQuerierFn(),
		config.NewSupplySessionQuerierFn(),
		supplyRelayMeter,
		supplyMiner,
		config.NewSupplyAccountQuerierFn(),
		config.NewSupplyBankQuerierFn(),
		config.NewSupplySupplierQuerierFn(),
		config.NewSupplyProofQueryClientFn(),
		config.NewSupplyRingClientFn(),
		supplyTxFactory,
		supplyTxContext,
		// RelayMiner always uses tx simulation for gas estimation (variable by tx).
		// Always use "auto" gas setting for RelayMiner.
		config.NewSupplySupplierClientsFn(signingKeyNames, cosmosflags.GasFlagAuto),
		newSupplyRelayAuthenticatorFn(signingKeyNames),
		newSupplyRelayerProxyFn(servicesConfigMap),
		newSupplyRelayerSessionsManagerFn(smtStorePath),
	}

	return config.SupplyConfig(ctx, cmd, supplierFuncs)
}

// supplyMiner constructs a Miner instance and returns a new depinject.Config with it supplied.
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

// supplyRelayMeter constructs a RelayMeter instance and returns a new depinject.Config with it supplied.
//
// - Supplies RelayMeter to the dependency injection config
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
func supplyRelayMeter(
	_ context.Context,
	deps depinject.Config,
	_ *cobra.Command,
) (depinject.Config, error) {
	rm, err := proxy.NewRelayMeter(deps)
	if err != nil {
		return nil, err
	}

	return depinject.Configs(deps, depinject.Supply(rm)), nil
}

// supplyTxFactory constructs a cosmostx.Factory instance and returns a new depinject.Config with it supplied.
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

// supplyTxContext constructs a transaction context and returns a new depinject.Config with it supplied.
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

// newSupplyRelayAuthenticatorFn returns a function which constructs a RelayAuthenticator and returns a new depinject.Config with it supplied.
//
// - Accepts signingKeyNames for authenticator setup
// - Returns a SupplierFn for dependency injection
//
// Parameters:
//   - signingKeyNames: List of signing key names
//
// Returns:
//   - config.SupplierFn: Supplier function for dependency injection
func newSupplyRelayAuthenticatorFn(
	signingKeyNames []string,
) config.SupplierFn {
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
// - Accepts servicesConfigMap for proxy setup
// - Returns a SupplierFn for dependency injection
//
// Parameters:
//   - servicesConfigMap: Map of services configuration
//
// Returns:
//   - config.SupplierFn: Supplier function for dependency injection
func newSupplyRelayerProxyFn(
	servicesConfigMap map[string]*relayerconfig.RelayMinerServerConfig,
) config.SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		relayerProxy, err := proxy.NewRelayerProxy(
			deps,
			proxy.WithServicesConfigMap(servicesConfigMap),
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
func newSupplyRelayerSessionsManagerFn(smtStorePath string) config.SupplierFn {
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
