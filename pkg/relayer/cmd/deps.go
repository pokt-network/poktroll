package cmd

import (
	"context"
	"fmt"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const defaultSessionCountForCacheClearing = 1

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
	if flagNodeGRPCURL != flags.OmittedDefaultFlagValue {
		parsedFlagNodeGRPCUrl, err := url.Parse(flagNodeGRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grpc query URL: %w", err)
		}
		queryNodeGRPCUrl = parsedFlagNodeGRPCUrl
	} else {
		// TODO_TECHDEBT(#1444): Delete this once #1444 is fixed and merged.
		_ = cmd.Flags().Set(cosmosflags.FlagGRPC, queryNodeGRPCUrl.String())
	}

	// Override config file's `QueryNodeUrl` and `txNodeRPCUrl` with `--node` flag if specified.
	if flagNodeRPCURL != flags.OmittedDefaultFlagValue {
		parsedFlagNodeRPCUrl, err := url.Parse(flagNodeRPCURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rpc query URL: %w", err)
		}
		queryNodeRPCUrl = parsedFlagNodeRPCUrl
		txNodeRPCUrl = parsedFlagNodeRPCUrl
	} else {
		// TODO_TECHDEBT(#1444): Delete this once #1444 is fixed and merged.
		_ = cmd.Flags().Set(cosmosflags.FlagNode, queryNodeRPCUrl.String())
	}

	signingKeyNames := uniqueSigningKeyNames(relayMinerConfig)
	servicesConfigMap := relayMinerConfig.Servers
	smtStorePath := relayMinerConfig.SmtStorePath

	supplierFuncs := []config.SupplierFn{
		config.NewSupplyLoggerFromCtx(ctx),
		config.NewSupplyCometClientFn(queryNodeRPCUrl),                    // leaf
		config.NewSupplyBlockClientFn(queryNodeRPCUrl),                    // leaf
		config.NewSupplyQueryClientContextFn(queryNodeGRPCUrl),            // leaf
		config.NewSupplyTxClientContextFn(queryNodeGRPCUrl, txNodeRPCUrl), // leaf

		// Setup params caches (clear on new sessions).
		// TODO_TECHDEBT(@red-0ne): Params cache should only be cleared when params change.
		// This is a temporary solution until we implement event-based cache clearing.
		// Tokenomics/gateway params not used in RelayMiner, so no cache needed.
		config.NewSupplyParamsCacheFn[sharedtypes.Params](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),   // leaf
		config.NewSupplyParamsCacheFn[apptypes.Params](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),      // leaf
		config.NewSupplyParamsCacheFn[sessiontypes.Params](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),  // leaf
		config.NewSupplyParamsCacheFn[prooftypes.Params](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),    // leaf
		config.NewSupplyParamsCacheFn[servicetypes.Params](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),  // leaf
		config.NewSupplyParamsCacheFn[suppliertypes.Params](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)), // leaf

		// Setup key-value caches for pocket types (clear on new sessions).
		config.NewSupplyKeyValueCacheFn[sharedtypes.Service](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),                // leaf
		config.NewSupplyKeyValueCacheFn[servicetypes.RelayMiningDifficulty](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)), // leaf
		config.NewSupplyKeyValueCacheFn[sharedtypes.Supplier](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),               // leaf
		config.NewSupplyKeyValueCacheFn[query.BlockHash](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),                    // leaf
		config.NewSupplyKeyValueCacheFn[prooftypes.Claim](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)),                   // leaf
		// Session querier returns *sessiontypes.Session, so cache must return pointers.
		config.NewSupplyKeyValueCacheFn[*sessiontypes.Session](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)), // leaf
		// Clear on new blocks to refresh application state after each block.
		// It is needed to ensure that Applications can upstake to continue being served.
		config.NewSupplyKeyValueCacheFn[apptypes.Application](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)), // leaf

		// Setup key-value for cosmos types
		// AccountI cache is used for caching accounts (clear on new sessions).
		config.NewSupplyKeyValueCacheFn[cosmostypes.AccountI](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)), // leaf
		// Balance cache is used for caching supplier operator account balances (clear on new sessions).
		config.NewSupplyKeyValueCacheFn[query.Balance](cache.WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)), // leaf

		config.NewSupplySharedQueryClientFn(),
		config.NewSupplyServiceQueryClientFn(),
		config.NewSupplyApplicationQuerierFn(),
		config.NewSupplySessionQuerierFn(),
		config.SupplyRelayMeterFn(relayMinerConfig.EnableOverServicing),
		config.SupplyMiner,
		config.NewSupplyAccountQuerierFn(),
		config.NewSupplyBankQuerierFn(),
		config.NewSupplySupplierQuerierFn(),
		config.NewSupplyProofQueryClientFn(),
		config.NewSupplyRingClientFn(),
		config.SupplyTxFactory,
		config.SupplyTxContext,
		// RelayMiner always uses tx simulation for gas estimation (variable by tx).
		// Always use "auto" gas setting for RelayMiner.
		config.NewSupplySupplierClientsFn(signingKeyNames, cosmosflags.GasFlagAuto),
		config.NewSupplyRelayAuthenticatorFn(signingKeyNames),
		config.NewSupplyRelayerProxyFn(servicesConfigMap, relayMinerConfig.Ping.Enabled),
		config.NewSupplyRelayerSessionsManagerFn(smtStorePath),
	}

	return config.SupplyConfig(ctx, cmd, supplierFuncs)
}
