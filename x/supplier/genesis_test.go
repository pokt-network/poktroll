package supplier_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/testutil/sample"
	sharedtypes "pocket/x/shared/types"
	"pocket/x/supplier"
	"pocket/x/supplier/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		SupplierList: []sharedtypes.Supplier{
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
			},
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SupplierKeeper(t)
	supplier.InitGenesis(ctx, *k, genesisState)
	got := supplier.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.SupplierList, got.SupplierList)
	// this line is used by starport scaffolding # genesis/test/assert
}
