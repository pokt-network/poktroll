package supplier

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/supplier"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState supplier.GenesisState) {
	// Set all the supplier
	for _, supplier := range genState.SupplierList {
		k.SetSupplier(ctx, supplier)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *supplier.GenesisState {
	genesis := supplier.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.SupplierList = k.GetAllSuppliers(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
