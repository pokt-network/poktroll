package migration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	migration "github.com/pokt-network/poktroll/x/migration/module"
	"github.com/pokt-network/poktroll/x/migration/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		MorseAccountState: &types.MorseAccountState{},
		MorseAccountClaimList: []types.MorseAccountClaim{
			{
				MorseSrcAddress: "0",
			},
			{
				MorseSrcAddress: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.MigrationKeeper(t)
	migration.InitGenesis(ctx, k, genesisState)
	got := migration.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.MorseAccountState, got.MorseAccountState)
	require.ElementsMatch(t, genesisState.MorseAccountClaimList, got.MorseAccountClaimList)
	// this line is used by starport scaffolding # genesis/test/assert
}
