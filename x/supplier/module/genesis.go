package supplier

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the supplier
	for _, elem := range genState.SupplierList {
		k.SetSupplier(ctx, elem)
	}
	// Set all the claim
	for _, elem := range genState.ClaimList {
		k.UpsertClaim(ctx, elem)
	}
	// Set all the proof
	for _, elem := range genState.ProofList {
		k.UpsertProof(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.SupplierList = k.GetAllSupplier(ctx)
	genesis.ClaimList = k.GetAllClaims(ctx)
	genesis.ProofList = k.GetAllProofs(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
