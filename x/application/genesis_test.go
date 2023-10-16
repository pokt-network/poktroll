package application_test

import (
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/testutil/sample"
	"pocket/x/application"
	"pocket/x/application/types"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TODO_IN_THIS_COMMIT(@olshansk): Update genesis config.yml and add a few more tests here
func TestGenesis(t *testing.T) {
	addr1 := sample.AccAddress()
	stake1 := sdk.NewCoin("upokt", sdk.NewInt(100))

	addr2 := sample.AccAddress()
	stake2 := sdk.NewCoin("upokt", sdk.NewInt(100))

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		ApplicationList: []types.Application{
			{
				Address: addr1,
				Stake:   &stake1,
			},
			{
				Address: addr2,
				Stake:   &stake2,
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
