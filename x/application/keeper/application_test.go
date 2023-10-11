package keeper_test

import (
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNApplication(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Application {
	items := make([]types.Application, n)
	for i := range items {
		items[i].Address = strconv.Itoa(i)

		keeper.SetApplication(ctx, items[i])
	}
	return items
}

func TestApplicationGet(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	items := createNApplication(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetApplication(ctx,
			item.Address,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestApplicationRemove(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	items := createNApplication(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveApplication(ctx,
			item.Address,
		)
		_, found := keeper.GetApplication(ctx,
			item.Address,
		)
		require.False(t, found)
	}
}

func TestApplicationGetAll(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	items := createNApplication(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllApplication(ctx)),
	)
}

// This is just a helper test so we can easily find what the address is of the application module
// since it's derived based off of the module name.
func TestApplicationModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "cosmos1rl3gjgzexmplmds3tq3r3yk84zlwdl6d0yhysr", moduleAddress.String())
}
