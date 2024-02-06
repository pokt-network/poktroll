package keeper_test

import (
	"context"
	"strconv"
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNService(keeper keeper.Keeper, ctx context.Context, n int) []types.Service {
	items := make([]types.Service, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetService(ctx, items[i])
	}
	return items
}

func TestServiceGet(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	items := createNService(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetService(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestServiceRemove(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	items := createNService(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveService(ctx,
			item.Index,
		)
		_, found := keeper.GetService(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestServiceGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	items := createNService(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllService(ctx)),
	)
}
