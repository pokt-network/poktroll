package application

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/x/application/keeper"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState application.GenesisState) {
	// Set all the application
	for _, app := range genState.ApplicationList {
		k.SetApplication(ctx, app)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *application.GenesisState {
	genesis := application.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.ApplicationList = k.GetAllApplications(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
