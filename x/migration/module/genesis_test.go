package migration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	migration "github.com/pokt-network/pocket/x/migration/module"
	"github.com/pokt-network/pocket/x/migration/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		MorseClaimableAccountList: []types.MorseClaimableAccount{
			{
				MorseSrcAddress: sample.MorseAddressHex(),
			},
			{
				MorseSrcAddress: sample.MorseAddressHex(),
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
