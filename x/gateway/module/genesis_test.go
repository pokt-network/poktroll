package gateway_test

import (
	"testing"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	gateway "github.com/pokt-network/pocket/x/gateway/module"
	"github.com/pokt-network/pocket/x/gateway/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		GatewayList: []types.Gateway{
			{
				Address: sample.AccAddress(),
			},
			{
				Address: sample.AccAddress(),
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.GatewayKeeper(t)
	gateway.InitGenesis(ctx, k, genesisState)
	got := gateway.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.GatewayList, got.GatewayList)
	// this line is used by starport scaffolding # genesis/test/assert
}
