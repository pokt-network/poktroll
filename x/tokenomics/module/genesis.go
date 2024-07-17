package tokenomics

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState tokenomics.GenesisState) {
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
func ExportGenesis(ctx context.Context, k keeper.Keeper) *tokenomics.GenesisState {
	genesis := tokenomics.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.RelayMiningDifficultyList = k.GetAllRelayMiningDifficulty(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
