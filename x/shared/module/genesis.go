package shared

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/shared/keeper"
	"github.com/pokt-network/poktroll/x/shared/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Errorf("unable to set params: %v: %w", genState.Params, err))
	}
	for _, paramsUpdate := range genState.ParamsHistory {
		params := paramsUpdate.Params
		if params == nil {
			continue
		}
		if err := k.SetParamsAtHeight(ctx, paramsUpdate.EffectiveHeight, *params); err != nil {
			panic(fmt.Errorf("unable to set params at height %d: %w", paramsUpdate.EffectiveHeight, err))
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.ParamsHistory = k.GetAllParamsHistory(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
