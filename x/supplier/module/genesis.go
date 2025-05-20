package supplier

import (
	"context"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the supplier
	for _, supplier := range genState.SupplierList {
		// Initialize genesis suppliers service config history with at least one entry
		if len(supplier.ServiceConfigHistory) == 0 {
			supplier.ServiceConfigHistory = make([]*sharedtypes.ServiceConfigUpdate, 0, len(supplier.Services))
			for _, service := range supplier.Services {
				supplierConfigUpdate := &sharedtypes.ServiceConfigUpdate{
					OperatorAddress:    supplier.OperatorAddress,
					Service:            service,
					ActivationHeight:   1,
					DeactivationHeight: 0,
				}
				supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, supplierConfigUpdate)
			}
		}

		k.SetAndIndexDehydratedSupplier(ctx, supplier)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.SupplierList = k.GetAllSuppliers(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
