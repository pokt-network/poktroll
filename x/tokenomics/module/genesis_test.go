package tokenomics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	tokenomicsmodule "github.com/pokt-network/poktroll/x/tokenomics/module"
)

func TestGenesis(t *testing.T) {
	genesisState := tokenomics.GenesisState{
		Params: tokenomics.DefaultParams(),

		RelayMiningDifficultyList: []tokenomics.RelayMiningDifficulty{
			{
				ServiceId: "0",
			},
			{
				ServiceId: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx, _, _ := keepertest.TokenomicsKeeperWithActorAddrs(t)
	tokenomicsmodule.InitGenesis(ctx, k, genesisState)
	got := tokenomicsmodule.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.RelayMiningDifficultyList, got.RelayMiningDifficultyList)
	// this line is used by starport scaffolding # genesis/test/assert
}
