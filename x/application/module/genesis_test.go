package application_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	appmodule "github.com/pokt-network/poktroll/x/application/module"
)

func TestGenesis(t *testing.T) {
	genesisState := application.GenesisState{
		Params: application.DefaultParams(),

		ApplicationList: []application.Application{
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				ServiceConfigs: []*shared.ApplicationServiceConfig{
					{
						Service: &shared.Service{Id: "svc1"},
					},
				},
			},
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(100)},
				ServiceConfigs: []*shared.ApplicationServiceConfig{
					{
						Service: &shared.Service{Id: "svc2"},
					},
				},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ApplicationKeeper(t)
	appmodule.InitGenesis(ctx, k, genesisState)
	got := appmodule.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.ApplicationList, got.ApplicationList)
	// this line is used by starport scaffolding # genesis/test/assert
}
