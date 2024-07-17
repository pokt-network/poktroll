package gateway_test

import (
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	gatewaymodule "github.com/pokt-network/poktroll/x/gateway/module"
)

func TestGenesis(t *testing.T) {
	genesisState := gateway.GenesisState{
		Params: gateway.DefaultParams(),

		GatewayList: []gateway.Gateway{
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
	gatewaymodule.InitGenesis(ctx, k, genesisState)
	got := gatewaymodule.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.GatewayList, got.GatewayList)
	// this line is used by starport scaffolding # genesis/test/assert
}
