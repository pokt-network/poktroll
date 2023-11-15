package config

import (
	"context"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/relayer"
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

// NewSupplyEventsQueryClientFn returns a new function which constructs an
// EventsQueryClient instance and returns a new depinject.Config which is
// supplied with the given deps and the new EventsQueryClient.
func NewSupplyEventsQueryClientFn(
	pocketNodeWebsocketURL string,
) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketURL)

		return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
	}
}

// NewSupplyBlockClientFn returns a function which constructs a BlockClient
// instance with the given nodeURL and returns a new depinject.Config which
// is supplied with the given deps and the new BlockClient.
func NewSupplyBlockClientFn(pocketNodeWebsocketURL string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		blockClient, err := block.NewBlockClient(ctx, deps, pocketNodeWebsocketURL)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(blockClient)), nil
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
		// Set --node flag to the --pocket-node for the client context
		// This flag is read by cosmosclient.GetClientQueryContext.
		err := cmd.Flags().Set(cosmosflags.FlagNode, pocketQueryNodeURL)
		if err != nil {
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
			relayer.QueryClientContext(queryClientCtx),
		))
		return deps, nil
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
