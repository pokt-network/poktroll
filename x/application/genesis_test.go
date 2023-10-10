package application_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/x/application"
	"pocket/x/application/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ApplicationKeeper(t)
	application.InitGenesis(ctx, *k, genesisState)
	got := application.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
