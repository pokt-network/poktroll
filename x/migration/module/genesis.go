package migration

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the morseClaimableAccount
	for _, morseClaimableAccounts := range genState.MorseClaimableAccountList {
		k.SetMorseClaimableAccount(ctx, morseClaimableAccounts)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.MorseClaimableAccountList = k.GetAllMorseClaimableAccounts(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
