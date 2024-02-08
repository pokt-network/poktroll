package supplier_test

import (
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/supplier/module"
	"github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		SupplierList: []types.Supplier{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		ClaimList: []types.Claim{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		ProofList: []types.Proof{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SupplierKeeper(t)
	supplier.InitGenesis(ctx, k, genesisState)
	got := supplier.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.SupplierList, got.SupplierList)
	require.ElementsMatch(t, genesisState.ClaimList, got.ClaimList)
	require.ElementsMatch(t, genesisState.ProofList, got.ProofList)
	// this line is used by starport scaffolding # genesis/test/assert
}
