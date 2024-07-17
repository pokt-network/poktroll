package proof

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/x/proof/keeper"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState proof.GenesisState) {
	// Set all the claim
	for _, claim := range genState.ClaimList {
		k.UpsertClaim(ctx, claim)
	}
	// Set all the proof
	for _, proof := range genState.ProofList {
		k.UpsertProof(ctx, proof)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *proof.GenesisState {
	genesis := proof.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.ClaimList = k.GetAllClaims(ctx)
	genesis.ProofList = k.GetAllProofs(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
