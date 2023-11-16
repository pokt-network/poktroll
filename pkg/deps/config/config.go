package config

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
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

// NewSupplyEventsQueryClientFn constructs an EventsQueryClient instance and returns
// a new depinject.Config which is supplied with the given deps and the new
// EventsQueryClient.
func NewSupplyEventsQueryClientFn(
	pocketNodeWebsocketUrl string,
) SupplierFn {
	return func(
		_ context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketUrl)

		return depinject.Configs(deps, depinject.Supply(eventsQueryClient)), nil
	}
}

// NewSupplyBlockClientFn returns a function with constructs a BlockClient instance
// with the given nodeURL and returns a new
// depinject.Config which is supplied with the given deps and the new
// BlockClient.
func NewSupplyBlockClientFn(pocketNodeWebsocketUrl string) SupplierFn {
	return func(
		ctx context.Context,
		deps depinject.Config,
		_ *cobra.Command,
	) (depinject.Config, error) {
		blockClient, err := block.NewBlockClient(ctx, deps, pocketNodeWebsocketUrl)
		if err != nil {
			return nil, err
		}

		return depinject.Configs(deps, depinject.Supply(blockClient)), nil
	}
}
