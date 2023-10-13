package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	sharedtypes "pocket/x/shared/types"
	"pocket/x/supplier/keeper"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNSupplier(keeper *keeper.Keeper, ctx sdk.Context, n int) []sharedtypes.Supplier {
	items := make([]sharedtypes.Supplier, n)
	for i := range items {
		items[i].Address = strconv.Itoa(i)

		keeper.SetSupplier(ctx, items[i])
	}
	return items
}

func TestSupplierGet(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNSupplier(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSupplier(ctx,
			item.Address,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestSupplierRemove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNSupplier(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSupplier(ctx,
			item.Address,
		)
		_, found := keeper.GetSupplier(ctx,
			item.Address,
		)
		require.False(t, found)
	}
}

func TestSupplierGetAll(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	items := createNSupplier(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSupplier(ctx)),
	)
}
