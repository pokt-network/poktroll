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

	nodeRPCURL, err := cmd.Flags().GetString(cosmosflags.FlagNode)
	if err != nil {
		return nil, err
	}

	nodeGRPCURL, err := cmd.Flags().GetString(cosmosflags.FlagGRPC)
	if err != nil {
		return nil, err
	}

	// Override config file's `QueryNodeGRPCUrl` with `--grpc-addr` flag if specified.
	if nodeGRPCURL != flags.OmittedDefaultFlagValue {
		if err = cmd.Flags().Set(cosmosflags.FlagGRPC, nodeGRPCURL); err != nil {
			return nil, err
		}

		parsedFlagNodeGRPCUrl, parseErr := url.Parse(nodeGRPCURL)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse grpc query URL: %w", parseErr)
		}
		queryNodeGRPCUrl = parsedFlagNodeGRPCUrl
	}

	// Override config file's `QueryNodeUrl` and `txNodeRPCUrl` with `--node` flag if specified.
	if nodeRPCURL != flags.DefaultNodeRPCURL {
		if err = cmd.Flags().Set(cosmosflags.FlagNode, nodeRPCURL); err != nil {
			return nil, err
		}

		parsedFlagNodeRPCUrl, err := url.Parse(nodeRPCURL)
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
		config.NewSupplyCometClientFn(queryNodeRPCUrl),                    // leaf
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
