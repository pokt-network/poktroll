package config

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/spf13/cobra"
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
