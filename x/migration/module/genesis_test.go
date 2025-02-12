package migration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	migration "github.com/pokt-network/poktroll/x/migration/module"
	"github.com/pokt-network/poktroll/x/migration/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		MorseClaimableAccountList: []types.MorseClaimableAccount{
			{
				Address: []byte(sample.MorseAddressHex()),
			},
			{
				Address: []byte(sample.MorseAddressHex()),
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
