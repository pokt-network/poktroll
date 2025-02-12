package migration_test

import (
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	migration "github.com/pokt-network/poktroll/x/migration/module"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		MorseClaimableAccountList: []types.MorseClaimableAccount{
			{
				Address: "0",
			},
			{
				Address: "1",
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

	require.ElementsMatch(t, genesisState.MorseClaimableAccountList, got.MorseClaimableAccountList)
	// this line is used by starport scaffolding # genesis/test/assert
}
