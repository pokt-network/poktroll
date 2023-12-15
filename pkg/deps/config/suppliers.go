package config

import (
	"context"
	"net/url"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	grpc "github.com/cosmos/gogoproto/grpc"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/sdk"
	"github.com/spf13/cobra"
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

// NewSupplyEventsQueryClientFn returns a new function which constructs an
// EventsQueryClient instance, with the given hostname converted into a websocket
// URL to subscribe to, and returns a new depinject.Config which is supplied
// with the given deps and the new EventsQueryClient.
func NewSupplyEventsQueryClientFn(queryHostname string) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		// Convert the host to a websocket URL
		pocketNodeWebsocketURL := sdk.HostToWebsocketURL(queryHost)
		eventsQueryClient := events.NewEventsQueryClient(pocketNodeWebsocketURL)

		return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
	}
}

// NewSupplyBlockClientFn returns a function which constructs a BlockClient
// instance and returns a new depinject.Config which is supplied with the
// given deps and the new BlockClient.
func NewSupplyBlockClientFn() SupplierFn {
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

// NewSupplyQueryClientContextFn returns a function with constructs a ClientContext
// instance with the given cmd and returns a new depinject.Config which is
// supplied with the given deps and the new ClientContext.
func NewSupplyQueryClientContextFn(pocketQueryNodeURL string) SupplierFn {
	return func(_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Temporarily store the flag's current value
		tmp := cosmosflags.FlagNode

		// Set --node flag to the pocketQueryNodeURL for the client context
		// This flag is read by cosmosclient.GetClientQueryContext.
		if err := cmd.Flags().Set(cosmosflags.FlagNode, pocketQueryNodeURL); err != nil {
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
		if err := cmd.Flags().Set(cosmosflags.FlagNode, tmp); err != nil {
			return nil, err
		}

		return deps, nil
	}
}

// NewSupplyTxClientContextFn returns a function with constructs a ClientContext
// instance with the given cmd and returns a new depinject.Config which is
// supplied with the given deps and the new ClientContext.
func NewSupplyTxClientContextFn(pocketTxNodeURL string) SupplierFn {
	return func(_ context.Context,
		deps depinject.Config,
		cmd *cobra.Command,
	) (depinject.Config, error) {
		// Temporarily store the flag's current value
		tmp := cosmosflags.FlagNode

		// Set --node flag to the pocketTxNodeURL for the client context
		// This flag is read by cosmosclient.GetClientTxContext.
		if err := cmd.Flags().Set(cosmosflags.FlagNode, pocketTxNodeURL); err != nil {
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
		if err := cmd.Flags().Set(cosmosflags.FlagNode, tmp); err != nil {
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
func NewSupplyPOKTRollSDKFn(
	queryNodeURL *url.URL,
	signingKeyName string,
) SupplierFn {
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

		config := &sdk.POKTRollSDKConfig{
			PrivateKey: privateKey,
			Deps:       deps,
		}

		poktrollSDK, err := sdk.NewPOKTRollSDK(ctx, config)
		if err != nil {
			return nil, err
		}

		// Supply the session querier to the provided deps
		return depinject.Configs(deps, depinject.Supply(poktrollSDK)), nil
	}
}
