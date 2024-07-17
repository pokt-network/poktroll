package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	servicemodule "github.com/pokt-network/poktroll/x/service/module"
)

func TestGenesis(t *testing.T) {
	genesisState := service.GenesisState{
		Params: service.DefaultParams(),

		ServiceList: []shared.Service{
			{
				Id:   "svc1",
				Name: "service one",
			},
			{
				Id:   "svc2",
				Name: "service two",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ServiceKeeper(t)
	servicemodule.InitGenesis(ctx, k, genesisState)
	got := servicemodule.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.ServiceList, got.ServiceList)
	// this line is used by starport scaffolding # genesis/test/assert
}
