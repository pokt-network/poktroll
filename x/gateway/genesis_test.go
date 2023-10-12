package gateway_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/x/gateway"
	"pocket/x/gateway/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		GatewayList: []types.Gateway{
			{
				Address: "0",
			},
			{
				Address: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.GatewayKeeper(t)
	gateway.InitGenesis(ctx, *k, genesisState)
	got := gateway.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.GatewayList, got.GatewayList)
	// this line is used by starport scaffolding # genesis/test/assert
}
