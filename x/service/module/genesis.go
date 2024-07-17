package service

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/x/service/keeper"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState service.GenesisState) {
	// Set all the service
	for _, service := range genState.ServiceList {
		k.SetService(ctx, service)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *service.GenesisState {
	genesis := service.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.ServiceList = k.GetAllServices(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
