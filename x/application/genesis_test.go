package application_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Please see `x/application/types/genesis_test.go` for extensive tests related to the validity of the genesis state.
func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		ApplicationList: []types.Application{
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
					{
						Service: &sharedtypes.Service{Id: "svc1"},
					},
				},
			},
			{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
					{
						Service: &sharedtypes.Service{Id: "svc2"},
					},
				},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ApplicationKeeper(t)
	application.InitGenesis(ctx, *k, genesisState)
	got := application.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.ApplicationList, got.ApplicationList)
	// this line is used by starport scaffolding # genesis/test/assert
}
