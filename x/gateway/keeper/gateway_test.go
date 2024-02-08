package keeper_test

import (
	"context"
	"strconv"
	"testing"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	"github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNGateway(keeper keeper.Keeper, ctx context.Context, n int) []types.Gateway {
	items := make([]types.Gateway, n)
	for i := range items {
		items[i].Address = strconv.Itoa(i)

		keeper.SetGateway(ctx, items[i])
	}
	return items
}

func TestGatewayGet(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	items := createNGateway(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetGateway(ctx,
			item.Address,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestGatewayRemove(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	items := createNGateway(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveGateway(ctx,
			item.Address,
		)
		_, found := keeper.GetGateway(ctx,
			item.Address,
		)
		require.False(t, found)
	}
}

func TestGatewayGetAll(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	items := createNGateway(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllGateway(ctx)),
	)
}
