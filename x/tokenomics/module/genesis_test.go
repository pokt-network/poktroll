package tokenomics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	tokenomics "github.com/pokt-network/poktroll/x/tokenomics/module"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		RelayMiningDifficultyList: []types.RelayMiningDifficulty{
			{
				ServiceId: "0",
			},
			{
				ServiceId: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx, _, _ := keepertest.TokenomicsKeeperWithActorAddrs(t, nil)
	tokenomics.InitGenesis(ctx, k, genesisState)
	got := tokenomics.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.RelayMiningDifficultyList, got.RelayMiningDifficultyList)
	// this line is used by starport scaffolding # genesis/test/assert
}
