package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNApplications(keeper keeper.Keeper, ctx context.Context, n int) []types.Application {
	apps := make([]types.Application, n)
	for i := range apps {
		apps[i].Address = strconv.Itoa(i)

		keeper.SetApplication(ctx, apps[i])
	}
	return apps
}

func TestApplicationGet(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	for _, app := range apps {
		foundApp, found := keeper.GetApplication(ctx, app.Address)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&app),
			nullify.Fill(&foundApp),
		)
	}
}
func TestApplicationRemove(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	for _, app := range apps {
		keeper.RemoveApplication(ctx, app.Address)
		_, found := keeper.GetApplication(ctx, app.Address)
		require.False(t, found)
	}
}

func TestApplicationGetAll(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(apps),
		nullify.Fill(keeper.GetAllApplications(ctx)),
	)
}
