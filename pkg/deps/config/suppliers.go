package config

import (
	"context"
	"fmt"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	grpc "github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/sdk"
)

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

// NewSupplyEventsQueryClientFn returns a new function which constructs an
// EventsQueryClient instance, with the given hostname converted into a websocket
// URL to subscribe to, and returns a new depinject.Config which is supplied
// with the given deps and the new EventsQueryClient.
func NewSupplyEventsQueryClientFn(queryNodeRPCURL *url.URL) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Convert the host to a websocket URL
		queryNodeWebsocketURL := queryNodeToWebsocketURL(queryNodeRPCURL)
		eventsQueryClient := events.NewEventsQueryClient(queryNodeWebsocketURL)

		return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
	}
}

// NewSupplyBlockClientFn returns a function which constructs a BlockClient
// instance with the given hostname, which is converted into a websocket URL,
// to listen for block events on-chain, and returns a new depinject.Config which
// is supplied with the given deps and the new BlockClient.
func NewSupplyBlockClientFn(queryNodeRPCURL *url.URL) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Convert the host to a websocket URL
		queryNodeWebsocketURL := queryNodeToWebsocketURL(queryNodeRPCURL)
		blockClient, err := block.NewBlockClient(ctx, deps, queryNodeWebsocketURL)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(blockClient)), nil
	}
}

// NewSupplyQueryClientContextFn returns a function with constructs a ClientContext
// instance with the given cmd and returns a new depinject.Config which is
// supplied with the given deps and the new ClientContext.
func NewSupplyQueryClientContextFn(queryNodeGRPCURL *url.URL) SupplierFn {
	return func(_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Temporarily store the flag's current value
		// TODO_TECHDEBT(#223) Retrieve value from viper instead, once integrated.
		tmpGRPC, err := cmd.Flags().GetString(cosmosflags.FlagGRPC)
		if err != nil {
			return nil, err
		}

		// Set --grpc-addr flag to the pocketQueryNodeURL for the client context
		// This flag is read by cosmosclient.GetClientQueryContext.
		// Cosmos-SDK is expecting a GRPC address formatted as <hostname>[:<port>],
		// so we only need to set the Host parameter of the URL to cosmosflags.FlagGRPC value.
		if err := cmd.Flags().Set(cosmosflags.FlagGRPC, queryNodeGRPCURL.Host); err != nil {
			return nil, err
		}

		// NB: Currently, the implementations of GetClientTxContext() and
		// GetClientQueryContext() are identical, allowing for their interchangeable
		// use in both querying and transaction operations. However, in order to support
		// independent configuration of client contexts for distinct querying and
		// transacting purposes. E.g.: transactions are dispatched to the sequencer
		// while queries are handled by a trusted full-node.
		queryClientCtx, err := cosmosclient.GetClientQueryContext(cmd)
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

// NewSupplyTxClientContextFn returns a function with constructs a ClientContext
// instance with the given cmd and returns a new depinject.Config which is
// supplied with the given deps and the new ClientContext.
func NewSupplyTxClientContextFn(txNodeGRPCURL *url.URL) SupplierFn {
	return func(_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Temporarily store the flag's current value
		// TODO_TECHDEBT(#223) Retrieve value from viper instead, once integrated.
		tmpGRPC, err := cmd.Flags().GetString(cosmosflags.FlagGRPC)
		if err != nil {
			return nil, err
		}

		// Set --node flag to the pocketTxNodeURL for the client context
		// This flag is read by cosmosclient.GetClientTxContext.
		// Cosmos-SDK is expecting a GRPC address formatted as <hostname>[:<port>],
		// so we only need to set the Host parameter of the URL to cosmosflags.FlagGRPC value.
		if err := cmd.Flags().Set(cosmosflags.FlagGRPC, txNodeGRPCURL.Host); err != nil {
			return nil, err
		}

		// NB: Currently, the implementations of GetClientTxContext() and
		// GetClientQueryContext() are identical, allowing for their interchangeable
		// use in both querying and transaction operations. However, in order to support
		// independent configuration of client contexts for distinct querying and
		// transacting purposes. E.g.: transactions are dispatched to the sequencer
		// while queries are handled by a trusted full-node.
		txClientCtx, err := cosmosclient.GetClientTxContext(cmd)
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

		return deps, nil
	}
}

// NewSupplyAccountQuerierFn returns a function with constructs an AccountQuerier
// instance with the required dependencies and returns a new depinject.Config which
// is supplied with the given deps and the new AccountQuerier.
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

// NewSupplyApplicationQuerierFn returns a function with constructs an
// ApplicationQuerier instance with the required dependencies and returns a new
// instance with the required dependencies and returns a new depinject.Config
// which is supplied with the given deps and the new ApplicationQuerier.
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

// NewSupplySessionQuerierFn returns a function which constructs a
// SessionQuerier instance with the required dependencies and returns a new
// depinject.Config which is supplied with the given deps and the new SessionQuerier.
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

// NewSupplySupplierQuerierFn returns a function which constructs a
// SupplierQuerier instance with the required dependencies and returns a new
// instance with the required dependencies and returns a new depinject.Config
// which is supplied with the given deps and the new SupplierQuerier.
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

// NewSupplyRingCacheFn returns a function with constructs a RingCache instance
// with the required dependencies and returns a new depinject.Config which is
// supplied with the given deps and the new RingCache.
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

// NewSupplyPOKTRollSDKFn returns a function which constructs a
// POKTRollSDK instance with the required dependencies and returns a new
// depinject.Config which is supplied with the given deps and the new POKTRollSDK.
func NewSupplyPOKTRollSDKFn(signingKeyName string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		var clientCtx cosmosclient.Context

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

		config := &sdk.POKTRollSDKConfig{PrivateKey: privateKey, Deps: deps}
		poktrollSDK, err := sdk.NewPOKTRollSDK(ctx, config)
		if err != nil {
			return nil, err
		}

		// Supply the session querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(poktrollSDK)), nil
	}
}

// queryNodeToWebsocketURL converts a query node URL to a CometBFT websocket URL.
// It takes the Host property of the queryNode URL and returns it as a websocket URL
// formatted as ws://<hostname>:<port>/websocket.
func queryNodeToWebsocketURL(queryNode *url.URL) string {
	return fmt.Sprintf("ws://%s/websocket", queryNode.Host)
}
