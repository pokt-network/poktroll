package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	service "github.com/pokt-network/poktroll/x/service/module"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		ServiceList: []sharedtypes.Service{
			{
				Id:   "svc1",
				Name: "service one",
			},
			{
				Id:   "svc2",
				Name: "service two",
			},
		},

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

	k, ctx := keepertest.ServiceKeeper(t)
	service.InitGenesis(ctx, k, genesisState)
	got := service.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.ServiceList, got.ServiceList)
	require.ElementsMatch(t, genesisState.RelayMiningDifficultyList, got.RelayMiningDifficultyList)
	// this line is used by starport scaffolding # genesis/test/assert
}
