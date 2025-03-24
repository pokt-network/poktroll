package service

import (
	"context"

	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the service
	for _, service := range genState.ServiceList {
		k.SetService(ctx, service)
	}
	// Set all the relayMiningDifficulty
	for _, difficulty := range genState.RelayMiningDifficultyList {
		k.SetRelayMiningDifficulty(ctx, difficulty)
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

	genesis.ServiceList = k.GetAllServices(ctx)
	genesis.RelayMiningDifficultyList = k.GetAllRelayMiningDifficulty(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
